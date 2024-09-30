package api

import (
    "errors"
    "bytes"
    "fmt"
    "io"
    "os"
    "crypto/aes"
    "crypto/cipher"
    "crypto/rand"

    "github.com/libp2p/go-libp2p/core/crypto"
    "github.com/libp2p/go-libp2p/core/host"
    "golang.org/x/crypto/bcrypt"
    "golang.org/x/crypto/pbkdf2"
    "golang.org/x/crypto/sha3"
)

type P2PService struct {}

var p2pHost *host.Host = nil;

func (s *P2PService) Login(username string, password string) (string, error) {
    if p2pHost != nil {
        return "", alreadyLoggedIn
    }
    //Open sqlite database
    db, err := dbOpen()
    if err != nil {
        return "", err
    }
    defer db.Close()

    var passwordHash []byte;
    var privateKeyIV []byte;
    var privateKeyCiphertext []byte;
    var privateKeySalt []byte;

    count, err := dbGetUser(db, username, &passwordHash, &privateKeyIV, &privateKeyCiphertext, &privateKeySalt)
    if err != nil {
        return "", err
    }
    if count == 0 {
        return "", invalidCredentials
    }

    passwordBytes := []byte(password)
    err = bcrypt.CompareHashAndPassword(passwordHash, passwordBytes)
    if err != nil {
        return "", invalidCredentials
    }

    //Recover key from password
    derivedKey := pbkdf2.Key(passwordBytes, privateKeySalt, 100000, 32, sha3.New256)

    block, err := aes.NewCipher(derivedKey)
    if err != nil {
        fmt.Fprintf(os.Stderr, "Failed to create AES cipher. %v\n", err)
        return "", internalError
    }

    mode := cipher.NewCBCDecrypter(block, privateKeyIV)

    privateKeyBytes := make([]byte, len(privateKeyCiphertext))
    mode.CryptBlocks(privateKeyBytes, privateKeyCiphertext)

    privateKey, err := crypto.UnmarshalPrivateKey(privateKeyBytes[:68])
    if err != nil {
        fmt.Fprintf(os.Stderr, "Failed to unmarshal private key. %v\n", err)
        return "", internalError
    }

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

    //Hash password with bcrypt
    passwordBytes := []byte(password)
    passwordHash, err := bcrypt.GenerateFromPassword(passwordBytes, bcrypt.DefaultCost)   
    if err != nil {
        fmt.Fprintf(os.Stderr, "Failed to generate hash password. %v\n", err)
        return "", internalError
    }

    //Username does not exist. Generate a rpc.public/private key pair for libp2p.
    privateKey, _, err := crypto.GenerateEd25519Key(rand.Reader)
    if err != nil {
        fmt.Fprintf(os.Stderr, "Failed to generate public/private key. %v\n", err)
        return "", internalError
    }

    privateKeyBytes, err := crypto.MarshalPrivateKey(privateKey)
    if err != nil {
        fmt.Fprintf(os.Stderr, "Failed to marshal private key. %v\n", err)
        return "", internalError
    }

    salt := make([]byte, 16)
    _, err = io.ReadFull(rand.Reader, salt)
    if err != nil {
        fmt.Fprintf(os.Stderr, "Failed to generate salt. %v\n", err)
        return "", internalError
    }

    //Create a encryption key from our password
    derivedKey := pbkdf2.Key(passwordBytes, salt, 100000, 32, sha3.New256)

    block, err := aes.NewCipher(derivedKey)
    if err != nil {
        fmt.Fprintf(os.Stderr, "Failed to create AES cipher. %v\n", err)
        return "", internalError
    }

    blockSize := block.BlockSize()
    padding := 0 
    if len(privateKeyBytes) % blockSize != 0 {
        padding = blockSize - (len(privateKeyBytes) % blockSize)
    }
    privateKeyCiphertext := make([]byte, len(privateKeyBytes) + padding)
    paddedPrivateKeyBytes := append(privateKeyBytes, bytes.Repeat([]byte{byte(padding)}, padding)...)

    privateKeyIV := make([]byte, blockSize)
    _, err = io.ReadFull(rand.Reader, privateKeyIV)
    if err != nil {
        fmt.Fprintf(os.Stderr, "Failed to generate initialization vector. %v\n", err)
        return "", internalError
    }

    //Encrypt and store results into database
    mode := cipher.NewCBCEncrypter(block, privateKeyIV)
    mode.CryptBlocks(privateKeyCiphertext, paddedPrivateKeyBytes)

    err = dbAddUser(db, username, passwordHash, privateKeyIV, privateKeyCiphertext, salt)
    if err != nil {
        return "", err
    }

    return "success", nil
}
