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
                               private_key_ciphertext TEXT, private_key_iv TEXT, private_key_salt TEXT, wallet_address TEXT)`

const createUploadTableQuery = `CREATE TABLE IF NOT EXISTS uploads
                              (id INTEGER PRIMARY KEY, peer_id TEXT, cid TEXT, filename TEXT, price FLOAT, size INTEGER, timestamp TEXT)`


const createDownloadTableQuery = `CREATE TABLE IF NOT EXISTS downloads
                                 (id INTEGER PRIMARY KEY, peer_id TEXT, provider_id TEXT, cid TEXT, filename TEXT, price FLOAT, size INTEGER, timestamp TEXT)`

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

    //Create uploads table if doesn't exist
    _, err = db.Exec(createUploadTableQuery)
    if err != nil {
        db.Close()
        log.Printf("Failed to create file table. %v\n", err)
        return db, internalError
    }

    //Create downloads table if doesn't exist
    _, err = db.Exec(createDownloadTableQuery)
    if err != nil {
        db.Close()
        log.Printf("Failed to create file table. %v\n", err)
        return db, internalError
    }



    return db, nil
}

func dbGetUser(db *sql.DB, username string, passwordHash *[]byte, privateKeyCiphertext *[]byte, 
                privateKeyIV *[]byte, privateKeySalt *[]byte, walletAddress *string) (int, error) {
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
    var tmpWalletAddress string

    err = db.QueryRow(`SELECT
                       password_hash, private_key_ciphertext, private_key_iv, private_key_salt, wallet_address 
                       FROM users WHERE username= ?`, username).
                       Scan(&passwordHashStr, &privateKeyCiphertextStr, &privateKeyIVStr, &privateKeySaltStr, &tmpWalletAddress)
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
    if walletAddress != nil {
        *walletAddress = tmpWalletAddress
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

    _, err = db.Exec(`INSERT INTO users
                     (username, password_hash, private_key_ciphertext, private_key_iv, private_key_salt, wallet_address)
                     VALUES (?, ?, ?, ?, ?, ?)`,
                     username,
                     hex.EncodeToString(passwordHash),
                     hex.EncodeToString(privateKeyCiphertext),
                     hex.EncodeToString(privateKeyIV),
                     hex.EncodeToString(privateKeySalt),
                     "")

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

func dbAddUpload(db *sql.DB, peerID string, cid string, filename string, price float64, size int64, timestamp string) error {
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
    err = db.QueryRow(`SELECT cid FROM uploads WHERE cid= ? AND peer_id= ?`, cid, peerID).Scan(&tmpCid)
    if err != nil {
        if err == sql.ErrNoRows {
            _, err = db.Exec(`INSERT INTO uploads (peer_id, cid, filename, price, size, timestamp) VALUES (?, ?, ?, ?, ?, ?)`,
                             peerID,
                             cid,
                             filename,
                             price,
                             size,
                             timestamp)
        }
    } else {
        _, err = db.Exec(`UPDATE uploads SET filename=?, price=?, size=? WHERE cid=? AND peer_id=?`,
                         filename,
                         price,
                         size,
                         cid,
                         peerID)
    }

    if err != nil {
        log.Printf("Failed to push file into database. %v\n", err)
        return internalError
    }

    return nil
}

func dbGetUploads(db *sql.DB, peerID string) ([]FileShareUpload, error) {
    var err error
    //Establish connection to database if doesn't exist
    if db == nil {
        db, err = dbOpen()
        if err != nil {
            return nil, err
        }
        defer db.Close()
    }

    files := []FileShareUpload{}

    rows, err := db.Query(`SELECT cid, filename, price, size, timestamp FROM uploads WHERE peer_id= ?`, peerID)
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
    var size int64
    var timestamp string
    for rows.Next() {
        err := rows.Scan(&cid, &filename, &price, &size, &timestamp)
        if err != nil {
            log.Printf("Failed to scan rows from SQL query. %v\n", err);
            return nil, internalError
        }
        files = append(files, FileShareUpload {
                                timestamp,
                                FileShareFile {
                                    FileShareMeta{
                                        Size: size,
                                        Price: price,
                                        Name: filename,
                                    },
                                    cid,
                                    peerID,
                                }})
    }

    return files, nil
}

func dbRemoveUpload(db *sql.DB, peerID string, cidStr string) error {
    var err error
    //Establish connection to database if doesn't exist
    if db == nil {
        db, err = dbOpen()
        if err != nil {
            return err
        }
        defer db.Close()
    }

    _, err = db.Exec(`DELETE FROM uploads WHERE peer_id=? AND cid=?`, peerID, cidStr)
    if err != nil {
        if err == sql.ErrNoRows {
            return nil
        }
        log.Printf("Failed to delete file from SQLITE database. %v\n", err)
        return internalError
    }

    return nil
}

func dbAddDownload(db *sql.DB, peerID string, providerID string, cid string, filename string, price float64, size int64, timestamp string) error {
    var err error
    // Establish connection to database if doesn't exist
    if db == nil {
        db, err = dbOpen()
        if err != nil {
            return err
        }
        defer db.Close()
    }
    _, err = db.Exec(`INSERT INTO downloads (peer_id, provider_id, cid, filename, price, size, timestamp) VALUES (?, ?, ?, ?, ?, ?, ?)`,
                     peerID,
                     providerID,
                     cid,
                     filename,
                     price,
                     size,
                     timestamp)

    if err != nil {
        log.Printf("Failed to push downloads into database. %v\n", err)
        return internalError
    }

    return nil
}

func dbGetDownloads(db *sql.DB, peerID string) ([]FileShareDownload, error) {
    var err error
    //Establish connection to database if doesn't exist
    if db == nil {
        db, err = dbOpen()
        if err != nil {
            return nil, err
        }
        defer db.Close()
    }

    files := []FileShareDownload{}

    rows, err := db.Query(`SELECT provider_id, cid, filename, price, size, timestamp FROM downloads WHERE peer_id= ?`, peerID)
    if err != nil {
        if err == sql.ErrNoRows {
            return files, nil
        }
        log.Printf("Failed to query SQLITE database. %v\n", err)
        return nil, internalError
    }

    var providerID string
    var cid string
    var filename string
    var price float64
    var size int64
    var timestamp string
    for rows.Next() {
        err := rows.Scan(&providerID, &cid, &filename, &price, &size, &timestamp)
        if err != nil {
            log.Printf("Failed to scan rows from SQL query. %v\n", err);
            return nil, internalError
        }
        files = append(files, FileShareDownload {
                        timestamp,
                        FileShareFile {
                            FileShareMeta {
                                Size: size,
                                Price: price,
                                Name: filename,
                            },
                            cid,
                            providerID,
                        }})
    }

    return files, nil
}

func dbSetWalletAddress(db *sql.DB, username string, walletAddress string) error {
     var err error
    //Establish connection to database if doesn't exist
    if db == nil {
        db, err = dbOpen()
        if err != nil {
            return err
        }
        defer db.Close()
    }

    _, err = db.Exec(`UPDATE users SET wallet_address=? WHERE username=?`, walletAddress, username)

    if err != nil {
        log.Printf("Failed to update wallet address. %v\n", err)
        return internalError
    }
    return nil
}
