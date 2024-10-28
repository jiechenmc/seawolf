# LibP2P Backend

To run the code:

Install Go:
```
wget https://go.dev/dl/go1.23.2.linux-amd64.tar.gz && \
sudo rm -rf /usr/local/go &&  \
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

Running the program will create a sqlite database file `seawolf_p2p.db` and Unix socket `seawolf_p2p.sock`. 

To forge Json-RPC requests from the command line:
```
{"jsonrpc": "2.0", "id": "1", "method": "p2p_<funcName(camelCase)>", "params": [...]} | nc -U seawolf_p2p.sock | jq .
```
