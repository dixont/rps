package main

import (
	"github.com/levenlabs/go-llog"
	"github.com/dgrijalva/jwt-go"
	"github.com/gorilla/websocket"
	"encoding/json"
	"time"
	"strconv"
	"net/http"
	"fmt"
)

type UserStateClaims struct {
	Username string `json:"username"`
	Gold     int `json:"gold"`
	// recommended having
	jwt.StandardClaims
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	//TODO: Check origin against app origin to. Or serve js app from this server.
	CheckOrigin: func(r *http.Request) bool { return true },
}

type Challenge struct {
	Username string `json:"username"`
	Gold     int `json:"gold"`
}

type queueHandler struct {
  queue []Challenge
}

func (handler *queueHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	userState := req.Context().Value("UserState").(UserStateClaims)
	llog.Info(fmt.Sprintf("%s making a bet of %s", userState.Username, strconv.Itoa(userState.Gold)))
	conn, err := upgrader.Upgrade(w, req, nil)
	if err != nil {
		llog.Error(err.Error())
		return
	}

	for {
		messageType, p, err := conn.ReadMessage()
		llog.Info("Message received: " + string(p))
		if err != nil {
			llog.Error("Failed in reading message from client: " + err.Error())
			return
		}
		err = conn.WriteMessage(messageType, p)
		if err != nil {
			llog.Error("Failed in writing message to client: " + err.Error())
			return
		}
	}
}

type RegisterRequest struct {
	Username string `json:"username"`
}



type registerHandler struct {
	Secret string
}

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

	expireTime := time.Now().Add(time.Hour * 1)

	claims := UserStateClaims{
		userRegistration.Username,
		100,
		jwt.StandardClaims{
			ExpiresAt: expireTime.Unix(),
			Issuer:    "localhost:9000",
		},
	}

	// Create a signed JWT with the user's state to validate their gold amount
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedToken, err := token.SignedString([]byte(handler.Secret))
	if err != nil {
		llog.Error("Failed to create and sign token because of error: " + err.Error())
	}
	llog.Info(signedToken)
	cookie := http.Cookie{Name: "Authorization", Value: "Bearer: " + signedToken, Expires: expireTime, HttpOnly: true, }
	http.SetCookie(w, &cookie)

	fmt.Fprintf(w, "User Successfully registered.")
}