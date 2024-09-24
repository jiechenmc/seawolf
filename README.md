Filesharing with rewards 💰💰💰

# To bootstrap the application
```bash
- ./bootstrap.sh 
```

# Getting Started
### Terminal 1 (BTCD)
```bash
- docker exec -it seawolf-btcwallet-1 bash
- btcctl --simnet --wallet --rpcuser=$btcdusername --rpcpass=$btcdpassword getnewaddress # SKIP IF ALREADY EXISTS
- btcctl --simnet --wallet --rpcuser=$btcdusername --rpcpass=$btcdpassword listreceivedbyaddress # SKIP IF ALREADY KNOW ADDRESS
- btcd --simnet --rpcuser=$btcdusername --rpcpass=$btcdpassword --miningaddr SZoGnna9NsjkZWusgFJ3DGirJpq22GqmES # Start btcd with a mining address
- cp /root/.btcd/rpc.cert /root/.btcwallet/btcd.cert # IMPORTANT
```

### Terminal 2 (BTCWALLET)
```bash
- docker exec -it seawolf-btcwallet-1 bash
- btcwallet --simnet -u rpcuser -P rpcpass --create  # Run this once to create a wallet; use the seed in discord
- btcwallet --simnet -u rpcuser -P rpcpass # Run this anytime u want to start the wallet
```

# Common Commands
```bash
- btcctl --simnet --rpcuser=$btcdusername --rpcpass=$btcdpassword generate 100
- btcctl --simnet --wallet --rpcuser=$btcdusername --rpcpass=$btcdpassword getbalance
```