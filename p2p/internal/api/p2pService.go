package api

import (
    "errors"
    "fmt"
    "os"

    "github.com/libp2p/go-libp2p/core/host"
)

type P2PService struct {}

var p2pHost *host.Host = nil;

func (s *P2PService) Login(username string, password string) (string, error) {
    if p2pHost != nil {
        return "", alreadyLoggedIn
    }
    db, err := dbOpen()
    if err != nil {
        return "", err
    }
    defer db.Close()

    var passwordHash []byte;
    var privateKeyCiphertext []byte;
    var privateKeyIV []byte;
    var privateKeySalt []byte;

    //Get user info from database
    count, err := dbGetUser(db, username, &passwordHash, &privateKeyCiphertext, &privateKeyIV, &privateKeySalt)
    if err != nil {
        return "", err
    }
    if count == 0 {
        return "", invalidCredentials
    }

    passwordBytes := []byte(password)
    if !cipherCompareHashAndPassword(passwordHash, passwordBytes) {
        return "", invalidCredentials
    }

    privateKey, err := cipherDecryptPrivateKey(passwordBytes, privateKeyCiphertext, privateKeyIV, privateKeySalt)

    //Create libp2p host with private key
    newHost, err := createLibp2pHost(&privateKey)
    if err != nil {
        fmt.Fprintf(os.Stderr, "Failed to create libp2p host. %v\n", err)
        return "", internalError
    }
    p2pHost = &newHost

    return "success",nil
}

func (s *P2PService) Register(username string, password string) (string, error) {
    db, err := dbOpen()
    if err != nil {
        return "", err
    }
    defer db.Close()

    //Query for username(return error if username already exists)
    count, err := dbGetUser(db, username, nil, nil, nil, nil)
    if count == 1 {
        return "", errors.New("Username already exists")
    }

    passwordBytes := []byte(password)
    //Hash password
    passwordHash, err := cipherEncryptPassword(passwordBytes)
    if err != nil {
        return "", err
    }

    //Username does not exist. Generate a key pair for libp2p.
    var privateKeyCiphertext []byte
    var privateKeyIV []byte
    var privateKeySalt []byte
    privateKeyCiphertext, err = cipherGenerateEncryptedPrivateKey(passwordBytes, &privateKeyIV, &privateKeySalt)
    if err != nil {
        return "", err
    }

    err = dbAddUser(db, username, passwordHash, privateKeyCiphertext, privateKeyIV, privateKeySalt)
    if err != nil {
        return "", err
    }

    return "success", nil
}
