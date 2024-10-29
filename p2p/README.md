# LibP2P Backend

To run the code:

Install Go:
```
wget https://go.dev/dl/go1.23.2.linux-amd64.tar.gz && 
sudo rm -rf /usr/local/go &&  
sudo tar -C /usr/local -xzf go1.23.2.linux-amd64.tar.gz && 
export PATH=$PATH:/usr/local/go/bin && 
echo "export PATH=$PATH:/usr/local/go/bin" >> ~/.bashrc
```

Build Program:
```
make
```

Run Program:
```
./seawolf_p2p
```

Running the program will create a sqlite database file `seawolf_p2p.db` and Unix socket `seawolf_p2p.sock`. 

To forge Json-RPC requests from the command line:
```
{"jsonrpc": "2.0", "id": "1", "method": "p2p_<funcName(camelCase)>", "params": [...]} | nc -U seawolf_p2p.sock | jq .
```


# API:

## p2p_register
Creates a new user

#### Parameters
```
Username: string - username for the new user
Password: string - password for the new user
Seed:     string - optional seed string for public/private key generation
```

#### Returns
```
string - "success"
```

## p2p_login
Logs in to an existing user

#### Parameters
```
Username: string - login username
Password: string - login password
```

#### Returns
```
string - peerID of the logged in user
```

### p2p_getPeers
Returns all known peers and their status

#### Parameters
```
None
```

#### Returns
```
[
{
    "peer_id":      string   - peerID of known peer
    "addrs":        []string - list of addresses of known peer
    "is_connected": bool     - bool indicating whether or not we're currently connected with this peer
},
...
]
```

## p2p_discoverFiles
Discovers file CIDs in the network

### Parameters
```
None
```

### Returns
```
[{
    "size":                 int     - size of file in bytes
    "data_cid":             string  - cid of file
    "providers": [
        {
            "peer_id":      string  - peer id of provider
            "price":        int     - price of the file
            "metadata_cid": string  - cid of metadata
            "file_name":    string  - name of file
        },
        ...
    ]
}]
```

## p2p_discoverFile
Discovers providers for a specific file given data CID or metadata CID

#### Parameters
```
CID: string - data CID or metadata CID
```
#### Returns
```
{
    "size":                 int     - size of file in bytes
    "data_cid":             string  - cid of file
    "providers": [
        {
            "peer_id":      string  - peer id of provider
            "price":        int     - price of the file
            "metadata_cid": string  - cid of metadata
            "file_name":    string  - name of file
        },
        ...
    ]
}
```

## p2p_putFile
Uploads a file

#### Parameters
```
FilePath: string - path to file
Price:    float  - price of the file
```
#### Returns
```
CID: string - metadata CID of file
```
### p2p_getFile
Downloads a file

#### Parameters
```
ProviderPeerID:   string - peer ID of the provider node
CID:              string - data or metadata CID
DownloadFilePath: string - destination file path
```
#### Returns
```
SessionID: int - session ID of the download. Can be used later for pausing/resuming
```
## p2p_getSession
Gets session stats

#### Parameters
```
SessionID: int - session ID of the requested session
```
#### Returns
```
{
    "session_id":  int    - session ID
    "req_cid":     string - CID of downloaded file
    "rx_bytes":    int    - bytes downloaded
    "paused":      int    - non-zero indicates paused
    "is_complete": bool   - whether session is complete
    "result":      int    - status code of complete session. Non-zero indicates error
}
```

## p2p_pause
Pauses a session

#### Parameters
```
SessionID: int - session ID of the requested session
```
#### Returns
```
None
```

## p2p_resume
Resumes a session

#### Parameters
```
SessionID: int - session ID of the requested session
```
#### Returns
```
None
```
