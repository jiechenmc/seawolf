Filesharing with rewards 💰💰💰

# To bootstrap the application

```bash
- ./bootstrap.sh
```

# Getting Started

### Terminal 1 (BTCD)

```bash
- docker exec -it seawolf-btcd-1 bash
- btcd -a 130.245.173.221:8333 # Start btcd with a mining address
```

### Terminal 2 (BTCWALLET)

```bash
- docker exec -it seawolf-btcd-1 bash
- btcwallet -u $btcdusername -P $btcdpassword --create  # Run this once to create a wallet; use the seed in discord
- btcwallet -u $btcdusername -P $btcdpassword # Run this anytime u want to start the wallet
```

### Terminal 3 (BTCCTL)

```bash
- bctl --wallet getnewaddress # SKIP IF ALREADY EXISTS
- bctl --wallet listreceivedbyaddress # SKIP IF ALREADY KNOW ADDRESS
```

# Common Commands

```bash
- bctl generate 100
- bctl --wallet getbalance
```
