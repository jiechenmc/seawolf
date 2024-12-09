package api

import (
    "log"
    "encoding/hex"
    "database/sql"
    _ "github.com/mattn/go-sqlite3"
)

const databasePath = "seawolf_p2p.db"
const createUserTableQuery = `CREATE TABLE IF NOT EXISTS users
                              (id INTEGER PRIMARY KEY, username TEXT UNIQUE, password_hash TEXT,
                               private_key_ciphertext TEXT, private_key_iv TEXT, private_key_salt TEXT)`

const createWalletTableQuery = `CREATE TABLE IF NOT EXISTS wallets
                                    (id INTEGER PRIMARY KEY, username TEXT UNIQUE, rpc_username TEXT,
                                     rpc_password_ciphertext TEXT, rpc_password_salt TEXT)`

const createFileTableQuery = `CREATE TABLE IF NOT EXISTS files
                              (id INTEGER PRIMARY KEY, peer_id TEXT UNIQUE, cid TEXT, filename TEXT, price FLOAT)`


func dbOpen() (*sql.DB, error) {
    db, err := sql.Open("sqlite3", databasePath)
    if err != nil {
        log.Printf("Failed to open SQLITE database. %v\n", err)
        return db, internalError
    }

    //Create user table if doesn't exist
    _, err = db.Exec(createUserTableQuery)
    if err != nil {
        db.Close()
        log.Printf("Failed to create user table. %v\n", err)
        return db, internalError
    }

    //Create wallets table if doesn't exist
    _, err = db.Exec(createWalletTableQuery)
    if err != nil {
        db.Close()
        log.Printf("Failed to create wallet table. %v\n", err)
        return db, internalError
    }

    //Create files table if doesn't exist
    _, err = db.Exec(createFileTableQuery)
    if err != nil {
        db.Close()
        log.Printf("Failed to create file table. %v\n", err)
        return db, internalError
    }



    return db, nil
}

func dbGetUser(db *sql.DB, username string, passwordHash *[]byte, privateKeyCiphertext *[]byte, privateKeyIV *[]byte, privateKeySalt *[]byte) (int, error) {
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

    err = db.QueryRow(`SELECT password_hash, private_key_ciphertext, private_key_iv, private_key_salt FROM users WHERE username= ?`, username).
                        Scan(&passwordHashStr, &privateKeyCiphertextStr, &privateKeyIVStr, &privateKeySaltStr)
    if err != nil {
        if err == sql.ErrNoRows {
            return 0, nil
        }
        log.Printf("Failed to query SQLITE database. %v\n", err)
        return 0, internalError
    }

    if passwordHash != nil {
        *passwordHash, err = hex.DecodeString(passwordHashStr)
    }
    if privateKeyCiphertext != nil {
        *privateKeyCiphertext, err = hex.DecodeString(privateKeyCiphertextStr)
    }
    if privateKeyIV != nil {
        *privateKeyIV, err = hex.DecodeString(privateKeyIVStr)
    }
    if privateKeySalt != nil {
        *privateKeySalt, err = hex.DecodeString(privateKeySaltStr)
    }

    if err != nil {
        log.Printf("Failed to decode hex strings. %v\n", err)
        return 0, internalError
    }

    return 1, nil
}

func dbAddUser(db *sql.DB, username string, passwordHash []byte, privateKeyCiphertext []byte, privateKeyIV []byte, privateKeySalt []byte) error {
    var err error
    //Establish connection to database if doesn't exist
    if db == nil {
        db, err = dbOpen()
        if err != nil {
            return err
        }
        defer db.Close()
    }

    _, err = db.Exec(`INSERT INTO users (username, password_hash, private_key_ciphertext, private_key_iv, private_key_salt) VALUES (?, ?, ?, ?, ?)`,
        username,
        hex.EncodeToString(passwordHash),
        hex.EncodeToString(privateKeyCiphertext),
        hex.EncodeToString(privateKeyIV),
        hex.EncodeToString(privateKeySalt))

    if err != nil {
        log.Printf("Failed to push user to database. %v\n", err)
        return internalError
    }

    return nil
}

func dbAddWallet(db *sql.DB, username string, rpcUsername string, rpcPasswordCiphertext []byte, rpcPasswordIV []byte, rpcPasswordSalt []byte) error {
    var err error
    //Establish connection to database if doesn't exist
    if db == nil {
        db, err = dbOpen()
        if err != nil {
            return err
        }
        defer db.Close()
    }

    _, err = db.Exec(`INSERT INTO wallets (username, rpcUsername, rpc_password_ciphertext, rpc_password_iv, rpc_password_salt) VALUES (?, ?, ?, ?, ?)`,
        username,
        rpcUsername,
        hex.EncodeToString(rpcPasswordCiphertext),
        hex.EncodeToString(rpcPasswordIV),
        hex.EncodeToString(rpcPasswordSalt))

    if err != nil {
        log.Printf("Failed to push wallet to database. %v\n", err)
        return internalError
    }

    return nil

}

func dbGetWallet(db *sql.DB, username string, rpcUsername *string, rpcPasswordCiphertext *[]byte, rpcPasswordIV *[]byte, rpcPasswordSalt *[]byte) (int, error) {
    var err error

    if rpcUsername == nil {
        return 0, internalError
    }

    //Establish connection to database if doesn't exist
    if db == nil {
        db, err = dbOpen()
        if err != nil {
            return 0, err
        }
        defer db.Close()
    }

    var rpcPasswordCiphertextStr string
    var rpcPasswordIVStr string
    var rpcPasswordSaltStr string

    err = db.QueryRow(`SELECT rpc_password_ciphertext, rpc_password_iv, rpc_password_salt FROM wallets WHERE username= ?`, username).
                        Scan(rpcUsername, &rpcPasswordCiphertextStr, &rpcPasswordIVStr, &rpcPasswordSaltStr)
    if err != nil {
        if err == sql.ErrNoRows {
            return 0, nil
        }
        log.Printf("Failed to query SQLITE database. %v\n", err)
        return 0, internalError
    }

    if rpcPasswordCiphertext != nil {
        *rpcPasswordCiphertext, err = hex.DecodeString(rpcPasswordCiphertextStr)
    }
    if rpcPasswordIV != nil {
        *rpcPasswordIV, err = hex.DecodeString(rpcPasswordIVStr)
    }
    if rpcPasswordSalt != nil {
        *rpcPasswordSalt, err = hex.DecodeString(rpcPasswordSaltStr)
    }
    return 1, nil
}

func dbAddFile(db *sql.DB, peerID string, cid string, filename string, price float64) error {
    var err error
    // Establish connection to database if doesn't exist
    if db == nil {
        db, err = dbOpen()
        if err != nil {
            return err
        }
        defer db.Close()
    }
    var tmpCid string
    // Check if cid already exists
    err = db.QueryRow(`SELECT cid FROM files WHERE cid= ? AND peer_id= ?`, cid, peerID).Scan(&tmpCid)
    if err != nil {
        if err == sql.ErrNoRows {
            _, err = db.Exec(`INSERT INTO files (peer_id, cid, filename, price) VALUES (?, ?, ?, ?)`,
                peerID,
                cid,
                filename,
                price)
        }
    } else {
        _, err = db.Exec(`UPDATE files SET filename=?, price=? WHERE cid=? AND peer_id=?`,
            filename,
            price,
            cid,
            peerID)
    }

    if err != nil {
        log.Printf("Failed to push file into database. %v\n", err)
        return internalError
    }

    return nil
}

func dbGetFiles(db *sql.DB, peerID string) (map[string]FileShareFileMeta, error) {
    var err error
    //Establish connection to database if doesn't exist
    if db == nil {
        db, err = dbOpen()
        if err != nil {
            return nil, err
        }
        defer db.Close()
    }

    var files = make(map[string]FileShareFileMeta)

    rows, err := db.Query(`SELECT cid, filename, price FROM users WHERE peer_id= ?`, peerID)
    if err != nil {
        if err == sql.ErrNoRows {
            return files, nil
        }
        log.Printf("Failed to query SQLITE database. %v\n", err)
        return nil, internalError
    }

    var cid string
    var filename string
    var price float64
    for rows.Next() {
        err := rows.Scan(&cid, &filename, &price)
        if err != nil {
            log.Printf("Failed to scan rows from SQL query. %v\n", err);
            return nil, internalError
        }
        files[cid] = FileShareFileMeta{ Size: 0, Price: price, Name: filename }
    }

    return files, nil
}
