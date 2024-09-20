Filesharing with rewards 💰💰💰

# To start the application
```bash
- docker compose up --build # -d to run in the background
```

# To create a wallet
```bash
- docker exec -it seawolf-btcwallet-1 bash
- btcwallet -u rpcuser -P rpcpass --create  # Run this once to create a wallet; use the seed in discord
- btcwallet -u rpcuser -P rpcpass # Run this anytime u want to start the wallet
```