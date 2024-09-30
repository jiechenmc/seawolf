package api

import (
    "fmt"
    "os"
    "encoding/hex"
    "database/sql"
    _ "github.com/mattn/go-sqlite3"
)

const databasePath = "seawolf_p2p.db"
const createTableQuery = `CREATE TABLE IF NOT EXISTS users
                            (id INTEGER PRIMARY KEY, username TEXT UNIQUE, password_hash TEXT,
                                private_key_iv TEXT, private_key_ciphertext TEXT, private_key_salt TEXT)`

func dbOpen() (*sql.DB, error) {
    db, err := sql.Open("sqlite3", databasePath)
    if err != nil {
        fmt.Fprintf(os.Stderr, "Failed to open SQLITE database. %v\n", err)
        return db, internalError
    }

    //Create user table if doesn't exist
    _, err = db.Exec(createTableQuery)
    if err != nil {
        db.Close()
        fmt.Fprintf(os.Stderr, "Failed to create user table. %v\n", err)
        return db, internalError
    }
    return db, nil
}

func dbGetUser(db *sql.DB, username string, passwordHash *[]byte, privateKeyIV *[]byte, privateKeyCiphertext *[]byte, privateKeySalt *[]byte) (int, error) {
    var err error
    //Establish connection to database if doesn't exist
    if db == nil {
        db, err = dbOpen()
        if err != nil {
            return 0, err
        }
        defer db.Close()
    }

    var privateKeyIVStr string
    var privateKeyCiphertextStr string
    var privateKeySaltStr string
    var passwordHashStr string

    err = db.QueryRow(`SELECT password_hash, private_key_iv, private_key_ciphertext, private_key_salt FROM users WHERE username= ?`, username).
                        Scan(&passwordHashStr, &privateKeyIVStr, &privateKeyCiphertextStr, &privateKeySaltStr)
    if err != nil {
        if err == sql.ErrNoRows {
            return 0, nil
        }
        fmt.Fprintf(os.Stderr, "Failed to query SQLITE database. %v\n", err)
        return 0, internalError
    }
    
    if passwordHash != nil {
        *passwordHash, err = hex.DecodeString(passwordHashStr)
    }
    if privateKeyIV != nil {
        *privateKeyIV, err = hex.DecodeString(privateKeyIVStr)
    }
    if privateKeySalt != nil {
        *privateKeySalt, err = hex.DecodeString(privateKeySaltStr)
    }
    if privateKeyCiphertext != nil {
        *privateKeyCiphertext, err = hex.DecodeString(privateKeyCiphertextStr)
    }

    if err != nil {
        fmt.Fprintf(os.Stderr, "Failed to decode hex strings. %v\n", err)
        return 0, internalError
    }

    return 1, nil
}

func dbAddUser(db *sql.DB, username string, passwordHash []byte, privateKeyIV []byte, privateKeyCiphertext []byte, privateKeySalt []byte) error {
    var err error
    //Establish connection to database if doesn't exist
    if db == nil {
        db, err = dbOpen()
        if err != nil {
            return err
        }
        defer db.Close()
    }

    _, err = db.Exec(`INSERT INTO users (username, password_hash, private_key_iv, private_key_ciphertext, private_key_salt) VALUES (?, ?, ?, ?, ?)`,
        username,
        hex.EncodeToString(passwordHash),
        hex.EncodeToString(privateKeyIV),
        hex.EncodeToString(privateKeyCiphertext),
        hex.EncodeToString(privateKeySalt))

    if err != nil {
        fmt.Fprintf(os.Stderr, "Failed to push user to database. %v\n", err)
        return internalError
    }

    return nil
}
