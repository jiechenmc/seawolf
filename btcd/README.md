Filesharing with rewards 💰💰💰

# To bootstrap the application

```bash
- ./bootstrap.sh
```

# Getting Started

### Terminal 1 (BTCD)

```bash
- docker exec -it seawolf-btcd-1 bash
- bctl --wallet getnewaddress # SKIP IF ALREADY EXISTS
- bctl --wallet listreceivedbyaddress # SKIP IF ALREADY KNOW ADDRESS
- btcd --$btcdnetwork --rpcuser=$btcdusername --rpcpass=$btcdpassword --miningaddr SUXxmVw5JWqSYC5syeCHBz15pEG2sFfxsk # Start btcd with a mining address
- cp /root/.btcd/rpc.cert /root/.btcwallet/btcd.cert # IMPORTANT; NEED TO BE DONE BEFORE FIRST STEP IF NOT FIRST TIME
```

### Terminal 2 (BTCWALLET)

```bash
- docker exec -it seawolf-btcd-1 bash
- btcwallet --$btcdnetwork -u $btcdusername -P $btcdpassword --create  # Run this once to create a wallet; use the seed in discord
- btcwallet --$btcdnetwork -u $btcdusername -P $btcdpassword # Run this anytime u want to start the wallet
```

# Common Commands

```bash
- bctl generate 100
- bctl --wallet getbalance
```
