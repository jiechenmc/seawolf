Filesharing with rewards 💰💰💰

# To bootstrap the application

```bash
- ./bootstrap.sh
```

# Getting Started

### Terminal 1 (BTCD)

```bash
- docker exec -it seawolf-btcd-1 bash
- bctl getnewaddress # SKIP IF ALREADY EXISTS
- bctl listreceivedbyaddress # SKIP IF ALREADY KNOW ADDRESS
- tmux new btcd --$btcdnetwork --rpcuser=$btcdusername --rpcpass=$btcdpassword --miningaddr SZoGnna9NsjkZWusgFJ3DGirJpq22GqmES # Start btcd with a mining address
- cp /root/.btcd/rpc.cert /root/.btcwallet/btcd.cert # IMPORTANT; NEED TO BE DONE BEFORE FIRST STEP IF NOT FIRST TIME
```

### Terminal 2 (BTCWALLET)

```bash
- docker exec -it seawolf-btcd-1 bash
- btcwallet --$btcdnetwork -u $btcdusername -P $btcdpassword --create  # Run this once to create a wallet; use the seed in discord
- tmux new btcwallet --$btcdnetwork -u $btcdusername -P $btcdpassword # Run this anytime u want to start the wallet
```

# Common Commands

```bash
- bctl generate 100
- bctl --wallet getbalance
```

# Seawolf Exchange

An Electron application with React and TypeScript

## Recommended IDE Setup

- [VSCode](https://code.visualstudio.com/) + [ESLint](https://marketplace.visualstudio.com/items?itemName=dbaeumer.vscode-eslint) + [Prettier](https://marketplace.visualstudio.com/items?itemName=esbenp.prettier-vscode)

## Project Setup

### Install

```bash
$ npm install
```

### Development

```bash
$ npm run dev
```

### Build

```bash
# For windows
$ npm run build:win

# For macOS
$ npm run build:mac

# For Linux
$ npm run build:linux
```
