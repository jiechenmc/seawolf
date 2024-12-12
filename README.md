# Seawolf Exchange

### Prerequisites

- Use a Linux/Unix system like Ubuntu
- See https://go.dev/doc/install to install `Go` 
- See https://docs.docker.com/engine/install/ to install `Docker` and its `docker-compose-plugin`. We are using Ubuntu and used this guide: https://docs.docker.com/engine/install/ubuntu/.


### Running the Electron App

```bash
npm install && npm run dev
```


### Running BTCD and BTCWALLET

Set the apporiate wallet seed in `docker-compose.yml` on line `12` before continuing. An empty wallet seed will result in the creation of a brand new wallet.

```bash
./bootstrap.sh
```

`bootstrap.sh` will compile p2p and create a container that will run `btcd`, `btcwallet` and another container that will run the `proxy`.

Please rerun `./bootstrap.sh` after any changes to the wallet seed.

### Running Proxy

The SOCKS5 proxy will be ran in a container in the setup by `bootstrap.sh` and traffic will be sent over TCP on port `8083` after you connect to a node acting as proxy.

### Running p2p Backend

```bash
./p2p/seawolf_p2p
```

Please rerun `./p2p/seawolf_p2p` for each user and for each session. So if you reopen the electron app on the same device, you need to restart the p2p backend. Please also delete the `seawolf_p2p.db` file if you plan on registering/logging in as the same user.