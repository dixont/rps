package main

import (
	"github.com/levenlabs/go-llog"
	"net/http"
	"os"
	"github.com/dgrijalva/jwt-go"
	"fmt"
	"context"
)

func validateUserState(secret string, authenticatedHandler http.Handler) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		cookie, err := req.Cookie("Authorization")

		if err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			fmt.Fprintf(w, "User must register before accessing this route.")
			return
		}

		tokenFromCookie := cookie.Value[8:]

		// Parse the token from the cookie
		token, err := jwt.ParseWithClaims(tokenFromCookie, &UserStateClaims{}, func(token *jwt.Token) (interface{}, error) {
			return []byte(secret), nil
		})

		if err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			fmt.Fprintf(w, "Has someone been tampering with their token? That's unfortunate...")
			return
		}

		// Grab the tokens claims and pass it into the original request
		claims, ok := token.Claims.(*UserStateClaims);

		if  ok && token.Valid {
			ctx := context.WithValue(req.Context(), "UserState", *claims)
			authenticatedHandler.ServeHTTP(w, req.WithContext(ctx))
		} else {
			w.WriteHeader(http.StatusUnauthorized)
			fmt.Fprintf(w, "Has someone been tampering with their token? That's unfortunate...")
			return
		}
	})
}

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

	http.Handle("/register", &registerHandler{Secret: secret})
	http.Handle("/challenge", validateUserState(secret, &queueHandler{}))

	err := http.ListenAndServe(":"+port, nil)

	if err != nil {
		llog.Fatal(err.Error())
	}
}
