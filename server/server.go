package main

import (
	"github.com/levenlabs/go-llog"
	"net/http"
	"github.com/gorilla/websocket"
	"os"
	"github.com/dgrijalva/jwt-go"
	"fmt"
	"encoding/json"
	"time"
)

type UserStateClaims struct {
	Username string `json:"username"`
	Gold     int `json:"gold"`
	// recommended having
	jwt.StandardClaims
}

var port = os.Getenv("PORT")
var secret = os.Getenv("SECRET")

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	//TODO: Check origin against app origin to. Or serve js app from this server.
	CheckOrigin: func(r *http.Request) bool { return true },
}

func handleLobby(w http.ResponseWriter, req *http.Request) {
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

func handleRegister(w http.ResponseWriter, req *http.Request) {
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
	signedToken, err := token.SignedString([]byte(secret))
	if err != nil {
		llog.Error("Failed to create and sign token because of error: " + err.Error())
	}
	llog.Info(signedToken)
	cookie := http.Cookie{Name: "Authorization", Value: "Bearer: " + signedToken, Expires: expireTime, HttpOnly: true, }
	http.SetCookie(w, &cookie)

	fmt.Fprintf(w, "User Successfully registered.")
}

func main() {

	// Configure server based on environment variables
	if len(port) == 0 {
		llog.Info("No port set in environment variables, setting it to 8000.")
		port = "8000"
	}

	if len(secret) == 0 {
		llog.Info("No secret set in environment variables, setting it to a default.")
		secret = "NotReallyButKindOfSecret"
	}

	logLevel := os.Getenv("LOG_LEVEL")

	if len(logLevel) == 0 {
		llog.Info("No log level set in environment variables, setting it to INFO.")
		llog.SetLevelFromString("INFO")
	} else {
		err := llog.SetLevelFromString(logLevel)
		if err != nil {
			llog.SetLevelFromString("INFO")
			llog.Error(err.Error())
			llog.Info("Defaulting log level to INFO.")
		}

	}

	http.HandleFunc("/register", handleRegister)
	http.HandleFunc("/lobby", handleLobby)

	err := http.ListenAndServe(":"+port, nil)

	if err != nil {
		llog.Fatal(err.Error())
	}
}
