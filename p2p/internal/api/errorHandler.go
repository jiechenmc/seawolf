package api

import (
    "errors"
)

var internalError = errors.New("Internal error")
var invalidCredentials = errors.New("Incorrect username or password")
var alreadyLoggedIn = errors.New("Already logged in")
var peerConnectionError = errors.New("Failed to connect to peer")
