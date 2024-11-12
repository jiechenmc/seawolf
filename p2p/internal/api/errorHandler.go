package api

import (
    "errors"
    // "context"
)

var internalError = errors.New("Internal error")
var invalidCredentials = errors.New("Error: Incorrect username or password")
var alreadyLoggedIn = errors.New("Error: Already logged in")
var notLoggedIn = errors.New("Error: Not logged in")
var peerConnectionError = errors.New("Error: Failed to connect to peer")
var invalidParams = errors.New("Error: Invalid parameter(s)")
var keyNotFound = errors.New("Error: Failed to find key")
var peerNotFound = errors.New("Error: Failed to find peer")
var timeoutError = errors.New("Error: Timed out")

var unexpectedResponse = errors.New("Error: Unexpected response")

//Fileshare
var failedToOpenFile = errors.New("Error: Failed to open file")
var sessionNotFound = errors.New("Error: Session not found")
var remoteSessionNotFound = errors.New("Error: Remote session not found")
var contentNotFound = errors.New("Error: Content not found")

//Chat
var chatNotFound = errors.New("Error: Chat not found")
var requestNotFound = errors.New("Error: Request not found")
var failedToSendMessage = errors.New("Error: Failed to send message")
var chatNotOngoing = errors.New("Error: Chat is not ongoing")

/* TODO
func mapError(err error) error {
    switch (err) {
        case context.DeadlineExceeded:
            return timeoutError
        case routing.ErrNotFound:
            return keyNotFound
        default:
            return internalError
    }
}*/
