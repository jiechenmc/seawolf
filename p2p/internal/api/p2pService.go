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
	"database/sql"
	"encoding/hex"
	
    "github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/host"
    _ "github.com/mattn/go-sqlite3"
    "golang.org/x/crypto/bcrypt"
	"golang.org/x/crypto/pbkdf2"
	"golang.org/x/crypto/sha3"
)

type P2PService struct {}

const databasePath = "seawolf_p2p.db"
const createTableQuery = `CREATE TABLE IF NOT EXISTS users
                            (id INTEGER PRIMARY KEY, username TEXT, password_hash TEXT,
                                iv TEXT, private_key_cipher TEXT, private_key_salt TEXT)`

var internalError = errors.New("Internal error")
var invalidCredentials = errors.New("Incorrect username or password")
var alreadyLoggedIn = errors.New("Already logged in")
var p2pHost *host.Host = nil;

func (s *P2PService) Login(username string, password string) (string, error) {
    if p2pHost != nil {
        return "", alreadyLoggedIn
    }
    //Open sqlite database
    db, err := sql.Open("sqlite3", databasePath)
    if err != nil {
        fmt.Fprintf(os.Stderr, "Failed to open SQLITE database. %v\n", err)
        return "", internalError
    }
    defer db.Close()
    
    //Create table if doesn't exist
	_, err = db.Exec(createTableQuery)
	if err != nil {
        fmt.Fprintf(os.Stderr, "Failed to create user table. %v\n", err)
        return "", internalError
	}
    
    //Query for password hash of the username
    var passwordHashStr string
    err = db.QueryRow(`SELECT password_hash FROM users WHERE username= ?`, username).Scan(&passwordHashStr)
    if err != nil {
        if err == sql.ErrNoRows {
            return "", invalidCredentials
        }
        fmt.Fprintf(os.Stderr, "Failed to query SQLITE database. %v\n", err)
        return "", internalError
    }
    passwordHash, err := hex.DecodeString(passwordHashStr)

    if err != nil {
        fmt.Fprintf(os.Stderr, "Failed to decode hex strings. %v\n", err)
        return "", internalError
    }

	passwordBytes := []byte(password)
    err = bcrypt.CompareHashAndPassword(passwordHash, passwordBytes)
    if err != nil {
        return "", invalidCredentials
    }

    var ivStr string
    var privateKeyCipherStr string
    var saltStr string
    err = db.QueryRow(`SELECT iv, private_key_cipher, private_key_salt FROM users WHERE username= ?`, username).Scan(&ivStr, &privateKeyCipherStr, &saltStr)
    if err != nil {
        fmt.Fprintf(os.Stderr, "Failed to query SQLITE database. %v\n", err)
        return "", internalError
    }
    
    iv, err := hex.DecodeString(ivStr)
    salt, err := hex.DecodeString(saltStr)
    privateKeyCipher, err := hex.DecodeString(privateKeyCipherStr)
    if err != nil {
        fmt.Fprintf(os.Stderr, "Failed to decode hex strings. %v\n", err)
        return "", internalError
    }
    
    //Recover key from password
	derivedKey := pbkdf2.Key(passwordBytes, salt, 100000, 32, sha3.New256)
       
    block, err := aes.NewCipher(derivedKey)
	if err != nil {
        fmt.Fprintf(os.Stderr, "Failed to create AES cipher. %v\n", err)
        return "", internalError
	}
    
    mode := cipher.NewCBCDecrypter(block, iv)

    privateKeyBytes := make([]byte, len(privateKeyCipher))
    mode.CryptBlocks(privateKeyBytes, privateKeyCipher)

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
    //Open sqlite database
    db, err := sql.Open("sqlite3", databasePath)
    if err != nil {
        fmt.Fprintf(os.Stderr, "Failed to open SQLITE database. %v\n", err)
        return "", internalError
    }
    defer db.Close()
    
    //Create table if doesn't exist
	_, err = db.Exec(createTableQuery);
	if err != nil {
        fmt.Fprintf(os.Stderr, "Failed to create user table. %v\n", err)
        return "", internalError
	}
    
    //Query for username(return error if username already exists)
    count := 0
    err = db.QueryRow(`SELECT COUNT(username) FROM users WHERE username= ?`, username).Scan(&count)
    if err != nil {
        fmt.Fprintf(os.Stderr, "Failed to query SQLITE database. %v\n", err)
        return "", internalError
    }
    
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
	ciphertext := make([]byte, len(privateKeyBytes) + padding)
    paddedPrivateKeyBytes := append(privateKeyBytes, bytes.Repeat([]byte{byte(padding)}, padding)...)

	iv := make([]byte, blockSize)
    _, err = io.ReadFull(rand.Reader, iv)
	if err != nil {
        fmt.Fprintf(os.Stderr, "Failed to generate nonce. %v\n", err)
        return "", internalError
	}

    //Encrypt and store results into database
	mode := cipher.NewCBCEncrypter(block, iv)
	mode.CryptBlocks(ciphertext, paddedPrivateKeyBytes)


	_, err = db.Exec(`INSERT INTO users (username, password_hash, iv, private_key_cipher, private_key_salt) VALUES (?, ?, ?, ?, ?)`,
        username,
        hex.EncodeToString(passwordHash),
		hex.EncodeToString(iv),
		hex.EncodeToString(ciphertext),
		hex.EncodeToString(salt))

	if err != nil {
        fmt.Fprintf(os.Stderr, "Failed to push rpc.user to database. %v\n", err)
        return "", internalError
	}

    return "success", nil
}
