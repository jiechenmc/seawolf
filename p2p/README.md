# LibP2P Backend

To run the code:

Install Go:
```
wget https://go.dev/dl/go1.23.2.linux-amd64.tar.gz && \
sudo rm -rf /usr/local/go && \
sudo tar -C /usr/local -xzf go1.23.2.linux-amd64.tar.gz && \
export PATH=$PATH:/usr/local/go/bin && \
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

Running the program will create a sqlite database file `seawolf_p2p.db`.

To forge Json-RPC requests from the command line:
```
curl -X POST \
     -H 'Content-Type: application/json' \
     -d '{ "jsonrpc":"2.0", "id":"<id>", "method":"p2p_<funcName(camelCase)", "params":[...]}' \
     http://localhost:8081/rpc
```


# API:

## NOTE: Return objects have an 'error' field if request has failed

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

## p2p_logout
Logs out

#### Parameters
```
None
```

#### Returns
```
string - "success"
```


## p2p_getPeers
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
## p2p_getFile
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
    "total_bytes": int    - size of file in bytes
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

## p2p_getUploads
Gets all uploaded files

#### Parameters
```
None
```
#### Returns
```
[{
    "size":         int     - size of file in bytes
    "price":        int     - price of the file
    "file_name":    string  - name of file
    "data_cid":     string  - cid of file
    "provider_id":  string  - peer id of provider
}]

```

## p2p_getDownloads
Gets all downloaded files

#### Parameters
```
None
```
#### Returns
```
[{
    "size":         int     - size of file in bytes
    "price":        int     - price of the file
    "file_name":    string  - name of file
    "data_cid":     string  - cid of file
    "provider_id":  string  - peer id of provider
}]

```

## p2p_deleteFile
Deletes an uploaded file

#### Parameters
```
CID: string - CID of file
```
#### Returns
```
None
```


## p2p_sendChatRequest
Sends a request to chat to a provider of a file

#### Parameters
```
peerID:     string - peer ID of the provider
fileCid:    string - CID of the file
```
#### Returns
```
{
    "request_id":   int     - request ID
    "peer_id":      string  - peer ID of the provider
    "file_cid":     string  - CID of the file
    "status":       string  - status of the request("pending", "accepted", or "declined")
}
```

## p2p_getIncomingChatRequests
Get incoming chat requests

#### Parameters
```
None
```
#### Returns
```
[
    {
        "request_id":   int     - request ID
        "peer_id":      string  - peer ID of the requester
        "file_cid":     string  - CID of the file
        "status":       string  - status of the request("pending", "accepted", or "declined")
    },
    ...
]
```

## p2p_getOutgoingChatRequests
Get outgoing chat requests

#### Parameters
```
None
```
#### Returns
```
[
    {
        "request_id":   int     - request ID
        "peer_id":      string  - peer ID of the provider
        "file_cid":     string  - CID of the file
        "status":       string  - status of the request("pending", "accepted", or "declined")
    },
    ...
]
```

## p2p_acceptChatRequest
Accepts an incoming chat request

#### Parameters
```
peerID:     string  - peer ID of the requester(request ID is only unique per requester)
requestID:  int     - request ID
```
#### Returns
```
{
    "chat_id":    int     - chat ID
    "buyer":      string  - peer ID of the buyer(requester)
    "seller":     string  - peer ID of the seller(this peer)
    "file_cid":   string  - CID of the file
    "status":     string  - status of the chat("ongoing", "finished", "timed out", or "error")
    "messages": [
        {
            "timestamp":    string - UTC timestamp of message
            "from":         string - peer ID of the sender
            "text":         string - message text
        },
        ...
    ]
}
```

## p2p_declineChatRequest
Declines an incoming chat request

#### Parameters
```
peerID:     string  - peer ID of the requester(request ID is only unique per requester)
requestID:  int     - request ID
```
#### Returns
```
None
```

## p2p_getChats
Gets all chats

#### Parameters
```
None
```
#### Returns
```
[
    {
        "chat_id":    int     - chat ID
        "buyer":      string  - peer ID of the buyer
        "seller":     string  - peer ID of the seller
        "file_cid":   string  - CID of the file
        "status":     string  - status of the chat("ongoing", "finished", "timed out", or "error")
        "messages": [
            {
                "timestamp":    string - UTC timestamp of message
                "from":         string - peer ID of the sender
                "text":         string - message text
            },
            ...
        ]
    },
    ...
]

```

## p2p_getChat
Get specific chat given chat id and remote peer id

#### Parameters
```
PeerID  string  - remote peer ID
ChatID  int     - chat ID
```
#### Returns
```
{
    "chat_id":    int     - chat ID
    "buyer":      string  - peer ID of the buyer
    "seller":     string  - peer ID of the seller
    "file_cid":   string  - CID of the file
    "status":     string  - status of the chat("ongoing", "finished", "timed out", or "error")
    "messages": [
        {
            "timestamp":    string - UTC timestamp of message
            "from":         string - peer ID of the sender
            "text":         string - message text
        },
        ...
    ]
}
```

## p2p_sendMessage
Sends a message within chat

#### Parameters
```
PeerID  string  - remote peer ID
ChatID  int     - chat ID
Text    string  - message text
```

#### Returns
```
{
    "timestamp":    string - UTC timestamp of message
    "from":         string - peer ID of the sender
    "text":         string - message text
}
```

## p2p_getMessages
Gets all messages within a chat

#### Parameters
```
PeerID  string  - remote peer ID
ChatID  int     - chat ID
```

#### Returns
```
[
    {
        "timestamp":    string - UTC timestamp of message
        "from":         string - peer ID of the sender
        "text":         string - message text
    },
    ...
]
```

## p2p_closeChat
Closes/ends an ongoing chat

#### Parameters
```
PeerID  string  - remote peer ID
ChatID  int     - chat ID
```

#### Returns
```
{
    "chat_id":    int     - chat ID
    "buyer":      string  - peer ID of the buyer
    "seller":     string  - peer ID of the seller
    "file_cid":   string  - CID of the file
    "status":     string  - status of the chat("ongoing", "finished", "timed out", or "error")
    "messages": [
        {
            "timestamp":    string - UTC timestamp of message
            "from":         string - peer ID of the sender
            "text":         string - message text
        },
        ...
    ]
}
```

## p2p_setWalletAddress
Sets the wallet address for the logged in user

#### Parameters
```
WalletAddress  string  - wallet address
```

#### Returns
```
None
```
