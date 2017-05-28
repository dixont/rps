package main

import (
	"bytes"
	"encoding/base64"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
)

func TestUserRegisterIsValid(t *testing.T) {
	assert := assert.New(t)

	for _, testCase := range []struct {
		body       string
		statusCode int
		response   string
	}{
		{`{}`, 400, "Request body must have a 'username' property."},
		{`{"username":"username"`, 400, "Failed to parse request because of error: unexpected EOF"},
	} {
		req := httptest.NewRequest("POST", "localhost:8000/register", bytes.NewReader([]byte(testCase.body)))
		w := httptest.NewRecorder()
		handleRegister := &registerHandler{Secret: "NotReallyButKindOfSecret"}
		handleRegister.ServeHTTP(w, req)

		resp := w.Result()
		body, _ := ioutil.ReadAll(resp.Body)

		assert.Equal(resp.StatusCode, testCase.statusCode, "Status codes should be equal")
		assert.Equal(string(body), testCase.response, "Response bodies should equal")
	}
}

func TestUserTokenGeneration(t *testing.T) {
	assert := assert.New(t)
	req := httptest.NewRequest("POST", "localhost:8000/register", bytes.NewReader([]byte(`{"username": "username"}`)))
	w := httptest.NewRecorder()

	handleRegister := &registerHandler{Secret: "NotReallyButKindOfSecret"}
	handleRegister.ServeHTTP(w, req)

	resp := w.Result()
	body, _ := ioutil.ReadAll(resp.Body)
	token := string(body)

	// Test the header and body of the token
	pieces := strings.Split(token, ".")
	tokenHeader, _ := base64.StdEncoding.DecodeString(pieces[0])
	assert.Equal(string(tokenHeader), `{"alg":"HS256","typ":"JWT"}`, "Token header should specify the correct algorithm and type")
	tokenBody, _ := base64.StdEncoding.DecodeString(pieces[1])
	assert.Regexp(regexp.MustCompile(`{"username":"username","gold":100,"exp":(\d*)`), string(tokenBody), "Token body should be as expected")
}

// Add unit tests for token generation/parsing?
