package api

import (
    "log"
    "errors"
    "context"
    "github.com/libp2p/go-libp2p/core/host"
    dht "github.com/libp2p/go-libp2p-kad-dht"
)

type P2PService struct {}

var p2pHost *host.Host = nil;
var kadDHT *dht.IpfsDHT = nil;

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
        log.Printf("Attempted login from unregistered user '%v'\n", username);
        return "", invalidCredentials
    }

    passwordBytes := []byte(password)
    if !cipherCompareHashAndPassword(passwordHash, passwordBytes) {
        log.Printf("Attempted login to user '%v' failed\n", username);
        return "", invalidCredentials
    }

    privateKey, err := cipherDecryptPrivateKey(passwordBytes, privateKeyCiphertext, privateKeyIV, privateKeySalt)

    //Create libp2p host with private key
    newHost, err := p2pCreateHost(&privateKey)
    if err != nil {
        return "", err
    }
    p2pHost = &newHost
    log.Printf("Successfully created libp2p host with peer ID: %v\n", (*p2pHost).ID());

    kadDHT, err = p2pCreateDHT(context.Background(), *p2pHost)
    if err != nil {
        //Destroy libp2p host
        closeErr := (*p2pHost).Close()
        if closeErr != nil {
            log.Fatal("Failed to clean up libp2p host after DHT creation failure")
        }
        p2pHost = nil
        return "", err
    }
    log.Printf("Successfully created DHT instance\n");


    //Connect to peer
    err = p2pConnectToPeer(*p2pHost, bootstrapNodeAddr);
    if err != nil {
        //Destroy libp2p host
        closeErr := (*p2pHost).Close()
        if closeErr != nil {
            log.Fatal("Failed to clean up libp2p host after peer connection failure")
        }
        p2pHost = nil
        return "", err
    }

    log.Printf("Successfully logged in user '%v'\n", username);
    return "success",nil
}

func (s *P2PService) Register(username string, password string, seed string) (string, error) {
    //Optional seed parameter for private key generation
    var seedBytes []byte = nil
    if seed != "" {
        seedBytes = []byte(seed)
    }

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
    privateKeyCiphertext, err = cipherGenerateEncryptedPrivateKey(passwordBytes, seedBytes, &privateKeyIV, &privateKeySalt)
    if err != nil {
        return "", err
    }

    err = dbAddUser(db, username, passwordHash, privateKeyCiphertext, privateKeyIV, privateKeySalt)
    if err != nil {
        return "", err
    }

    log.Printf("Successfully registered user '%v'\n", username);
    return "success", nil
}
