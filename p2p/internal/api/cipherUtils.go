package api

import (
    "log"
    "io"
    "bytes"
    "crypto/aes"
    "crypto/cipher"
    "crypto/rand"
    "crypto/sha256"

    "github.com/libp2p/go-libp2p/core/crypto"
    "golang.org/x/crypto/bcrypt"
    "golang.org/x/crypto/pbkdf2"
    "golang.org/x/crypto/sha3"
)

func cipherEncryptPassword(passwordBytes []byte) ([]byte, error) {
    passwordHash, err := bcrypt.GenerateFromPassword(passwordBytes, bcrypt.DefaultCost)
    if err != nil {
        log.Printf("Failed to generate hash password. %v\n", err)
        return nil, internalError
    }
    return passwordHash, err
}

func cipherCompareHashAndPassword(passwordHash []byte, passwordBytes []byte) bool {
    err := bcrypt.CompareHashAndPassword(passwordHash, passwordBytes)
    if err != nil {
        return false
    }
    return true
}

func cipherGenerateEncryptedPrivateKey(passwordBytes []byte, seedBytes []byte, privateKeyIV *[]byte, privateKeySalt *[]byte) ([]byte, error) {
    var reader io.Reader
    //If seed is provided, use this seed to generate private key
    if seedBytes != nil {
        hash := sha256.Sum256(seedBytes)
        reader = bytes.NewReader(hash[:])
    } else {
        reader = rand.Reader
    }
    //Generate a public/private key pair
    privateKey, _, err := crypto.GenerateEd25519Key(reader)
    if err != nil {
        log.Printf("Failed to generate public/private key. %v\n", err)
        return nil, internalError
    }

    //Serialize private key to raw bytes
    privateKeyBytes, err := crypto.MarshalPrivateKey(privateKey)
    if err != nil {
        log.Printf("Failed to marshal private key. %v\n", err)
        return nil, internalError
    }

    //Generate a salt for encryption key
    *privateKeySalt = make([]byte, 16)
    _, err = io.ReadFull(rand.Reader, *privateKeySalt)
    if err != nil {
        log.Printf("Failed to generate salt. %v\n", err)
        return nil, internalError
    }

    //Create a encryption key from our password
    derivedKey := pbkdf2.Key(passwordBytes, *privateKeySalt, 100000, 32, sha3.New256)

    block, err := aes.NewCipher(derivedKey)
    if err != nil {
        log.Printf("Failed to create AES cipher. %v\n", err)
        return nil, internalError
    }

    blockSize := block.BlockSize()
    padding := 0
    if len(privateKeyBytes) % blockSize != 0 {
        padding = blockSize - (len(privateKeyBytes) % blockSize)
    }
    privateKeyCiphertext := make([]byte, len(privateKeyBytes) + padding)
    paddedPrivateKeyBytes := append(privateKeyBytes, bytes.Repeat([]byte{byte(padding)}, padding)...)

    //Generate initialization vector
    *privateKeyIV = make([]byte, blockSize)
    _, err = io.ReadFull(rand.Reader, *privateKeyIV)
    if err != nil {
        log.Printf("Failed to generate initialization vector. %v\n", err)
        return nil, internalError
    }

    //Encrypt and store results into database
    mode := cipher.NewCBCEncrypter(block, *privateKeyIV)
    mode.CryptBlocks(privateKeyCiphertext, paddedPrivateKeyBytes)

    return privateKeyCiphertext, nil
}


func cipherDecryptPrivateKey(passwordBytes []byte, privateKeyCiphertext []byte, privateKeyIV []byte, privateKeySalt []byte) (crypto.PrivKey, error) {
    //Recover key from password
    derivedKey := pbkdf2.Key(passwordBytes, privateKeySalt, 100000, 32, sha3.New256)
    block, err := aes.NewCipher(derivedKey)
    if err != nil {
        log.Printf("Failed to create AES cipher. %v\n", err)
        return nil, internalError
    }
    //Decrypt private key ciphertext
    mode := cipher.NewCBCDecrypter(block, privateKeyIV)

    privateKeyBytes := make([]byte, len(privateKeyCiphertext))
    mode.CryptBlocks(privateKeyBytes, privateKeyCiphertext)

    privateKey, err := crypto.UnmarshalPrivateKey(privateKeyBytes[:68])
    if err != nil {
        log.Printf("Failed to unmarshal private key. %v\n", err)
        return nil, internalError
    }
    return privateKey, nil
}
