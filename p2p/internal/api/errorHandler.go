package api

import (
    "errors"
)

var internalError = errors.New("Internal error")
var invalidCredentials = errors.New("Error: Incorrect username or password")
var alreadyLoggedIn = errors.New("Error: Already logged in")
var notLoggedIn = errors.New("Error: Not logged in")
var peerConnectionError = errors.New("Error: Failed to connect to peer")
var invalidParams = errors.New("Error: Invalid parameter(s)")
var keyNotFound = errors.New("Error: Failed to find key")
var peerNotFound = errors.New("Error: Failed to find peer")
