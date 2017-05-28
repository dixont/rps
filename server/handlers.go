package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/dgrijalva/jwt-go"
	"github.com/gorilla/websocket"
	"github.com/levenlabs/go-llog"
	"net/http"
	"strconv"
	"time"
)

type UserStateClaims struct {
	Username string `json:"username"`
	Gold     int    `json:"gold"`
	// recommended having
	jwt.StandardClaims
}

type Challenge struct {
	Username         string `json:"username"`
	InitialGold      int
	Gold             int    `json:"gold"`
	Throw            string `json:"throw"`
	Token            string `json:"token"`
	MessageType      int
	SocketConnection *websocket.Conn
}

type queueHandler struct {
	Queue    []Challenge
	Secret   string
	Upgrader websocket.Upgrader
}

type ChallengeResponse struct {
	Outcome string `json:"outcome"`
	Gold    int    `json:"gold"`
	Token   string `json:"token"`
	Error   string `json:"error"`
	Opposer string `json:"opposer"`
}

/**
 * Handler for the challenge queue.
 * Puts challenges into a queue, and if the length of the queue is greater than 0, the
 * newest challenge request challenges the first in the queue and the queue shifts 1.
 *
 *
 */
func (handler *queueHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	conn, err := handler.Upgrader.Upgrade(w, req, nil)
	if err != nil {
		llog.Error(err.Error())
		return
	}
	for {
		messageType, p, _ := conn.ReadMessage()
		bodyDecoder := json.NewDecoder(bytes.NewReader(p))

		var userChallenge Challenge
		err = bodyDecoder.Decode(&userChallenge)

		//Ensure the request has a token
		if userChallenge.Token == "" {
			llog.Error(fmt.Sprintf("User making request with no token"))
			res, _ := json.Marshal(&ChallengeResponse{Error: "You must register and make the challenge with your state token."})
			conn.WriteMessage(messageType, res)
		}
		userState, ok := parseToken(userChallenge.Token, handler.Secret)
		llog.Info(strconv.Itoa(userChallenge.Gold))
		llog.Info(strconv.Itoa(userState.Gold))
		if !ok { // Error in parsing user token
			llog.Error(fmt.Sprintf("User %s making request with potentially bogus token", userChallenge.Username))
			res, _ := json.Marshal(&ChallengeResponse{Error: "Trying to make request with bad token?"})
			conn.WriteMessage(messageType, res)
		} else if userChallenge.Gold > userState.Gold { // Disallow user from betting more gold than their state token has
			llog.Error(fmt.Sprintf("%s trying to bet %s, more money than he has (%s)!", userState.Username, strconv.Itoa(userChallenge.Gold), strconv.Itoa(userState.Gold)))
			res, _ := json.Marshal(&ChallengeResponse{Error: "Trying to bet more than you can!"})
			conn.WriteMessage(messageType, res)
		} else if userChallenge.Gold < 1 { //Disallow user to bet less than 1 gold
			llog.Error(fmt.Sprintf("%s trying to bet less than 1 (%s)! That's inconceivable!", userState.Username, strconv.Itoa(userChallenge.Gold)))
			res, _ := json.Marshal(&ChallengeResponse{Error: "You can't bet less than 1"})
			conn.WriteMessage(messageType, res)
		} else {
			// Initialize more challenge information with user state, then handle the challenge if there is another in the queue.
			userChallenge.Username = userState.Username
			userChallenge.InitialGold = userState.Gold
			userChallenge.SocketConnection = conn
			userChallenge.MessageType = messageType
			llog.Debug(fmt.Sprintf("%s making a bet of %s with throw %s", userChallenge.Username, strconv.Itoa(userChallenge.Gold), userChallenge.Throw))

			if len(handler.Queue) > 0 {
				opposingChallenge := handler.Queue[0]
				handler.Queue = handler.Queue[1:]
				handleChallenges(userChallenge, opposingChallenge, handler.Secret)
			} else {
				handler.Queue = append(handler.Queue, userChallenge)
			}
			break
		}
	}
}

func handleChallenges(challengeOne Challenge, challengeTwo Challenge, secret string) {

	var victor Challenge
	var loser Challenge
	victorState := "WIN"
	loserState := "LOSS"
	// if it is a tie, set both challenge golds to 0, otherwise, set the victors and losers appropriately and respond with winnings
	if challengeOne.Throw == challengeTwo.Throw {
		llog.Info(fmt.Sprintf("User %s and User %s draw when both throwing %s. Both keep their original ammount.", challengeOne.Username, challengeTwo.Username, challengeOne.Throw))
		victor = challengeOne
		victor.Gold = 0
		loser = challengeTwo
		loser.Gold = 0
		victorState = "TIE"
		loserState = "TIE"
	} else if (challengeOne.Throw == "r" && challengeTwo.Throw == "s") ||
		(challengeOne.Throw == "p" && challengeTwo.Throw == "r") ||
		(challengeOne.Throw == "s" && challengeTwo.Throw == "p") {
		victor = challengeOne
		loser = challengeTwo
	} else {
		loser = challengeOne
		victor = challengeTwo
	}

	if loserState != "TIE" {
		llog.Info(fmt.Sprintf("User %s (threw %s) beat User %s (threw %s), Victor gets %s", victor.Username, victor.Throw, loser.Username, loser.Throw, strconv.Itoa(victor.Gold+loser.Gold)))
	}

	// Return users their new state and gold amount, then close the connection
	victorRes, _ := json.Marshal(&ChallengeResponse{
		Token:   generateToken(victor.Username, victor.InitialGold+loser.Gold, secret),
		Gold:    victor.InitialGold + loser.Gold,
		Outcome: victorState,
		Opposer: loser.Username,
	})
	victor.SocketConnection.WriteMessage(victor.MessageType, victorRes)

	loserRes, _ := json.Marshal(&ChallengeResponse{
		Token:   generateToken(loser.Username, loser.InitialGold-loser.Gold, secret),
		Gold:    loser.InitialGold - loser.Gold,
		Outcome: loserState,
		Opposer: victor.Username,
	})
	loser.SocketConnection.WriteMessage(victor.MessageType, loserRes)

	victor.SocketConnection.Close()
	loser.SocketConnection.Close()
}

type RegisterRequest struct {
	Username string `json:"username"`
}

type registerHandler struct {
	Secret string
}

/**
 * Generate a state token for the user in the form of a JWT.
 */
func (handler *registerHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	if req.Method != "POST" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		fmt.Fprintf(w, "Method %s not allowed on route /register. Must be POST.", req.Method)
		return
	}

	bodyDecoder := json.NewDecoder(req.Body)
	var userRegistration RegisterRequest
	err := bodyDecoder.Decode(&userRegistration)

	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "Failed to parse request because of error: %s", err.Error())
		return
	}

	if userRegistration.Username == "" {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "Request body must have a 'username' property.")
		return
	}
	signedToken := generateToken(userRegistration.Username, 100, handler.Secret)

	fmt.Fprintf(w, signedToken)
}

/**
 * Generate JWT based on a username, gold, and secret
 */
func generateToken(username string, gold int, secret string) string {

	expireTime := time.Now().Add(time.Hour * 1)

	claims := UserStateClaims{
		username,
		gold,
		jwt.StandardClaims{
			ExpiresAt: expireTime.Unix(),
		},
	}

	// Create a signed JWT with the user's state to validate their gold amount
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedToken, err := token.SignedString([]byte(secret))
	if err != nil {
		llog.Error("Failed to create and sign token because of error: " + err.Error())
	}
	return signedToken
}

/**
 * Parse a token returning the UserState if properly parsed
 */
func parseToken(tokenFromRequest string, secret string) (*UserStateClaims, bool) {

	// Parse the token based on a secret
	token, err := jwt.ParseWithClaims(tokenFromRequest, &UserStateClaims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(secret), nil
	})

	if err != nil {
		return nil, false
	}

	// Grab the tokens claims and return it
	claims, ok := token.Claims.(*UserStateClaims)

	if ok && token.Valid {
		return claims, true
	}
	return nil, false
}
