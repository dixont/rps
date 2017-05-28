package main

import (
	"github.com/gorilla/websocket"
	"github.com/levenlabs/go-llog"
	"github.com/rs/cors"
	"net/http"
	"os"
)

func main() {

	// Configure server based on environment variables
	port := os.Getenv("PORT")
	if len(port) == 0 {
		llog.Info("No port set in environment variables, setting it to 8000.")
		port = "8000"
	}

	secret := os.Getenv("SECRET")
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

	// Set up a cors handler as middle ware to allow for easy communication between
	// two different serving ports.
	c := cors.Default()
	mux := http.NewServeMux()
	mux.Handle("/register", c.Handler(&registerHandler{Secret: secret}))
	mux.Handle("/challenge", c.Handler(&queueHandler{Secret: secret, Upgrader: websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		//TODO: Check origin against app origin to. Or serve js app from this server.
		CheckOrigin: func(r *http.Request) bool { return true },
	}}))
	handler := cors.Default().Handler(mux)

	err := http.ListenAndServe(":"+port, handler)

	if err != nil {
		llog.Fatal(err.Error())
	}
}
