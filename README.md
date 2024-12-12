# Seawolf Exchange

### Prerequisites

- See https://go.dev/doc/install to install `Go` 
- See https://docs.docker.com/engine/install/ to install `Docker` and its `docker-compose-plugin`. We are using Ubuntu and used this guide: https://docs.docker.com/engine/install/ubuntu/.


### Running the Electron App

```bash
$ npm install && npm run dev
```

The following steps assume you are on a `Linux/Unix` like system and you have `Go` and `Docker` installed and that you are at the root of the project. 


### Running BTCD and BTCWALLET

Set the apporiate wallet seed in `docker-compose.yml` on line `12` before continuing. An empty wallet seed will result in the creation of a brand new wallet.

```bash
$ ./bootstrap.sh
```

`bootstrap.sh` will compile p2p and create a container that will run `btcd`, `btcwallet` and another container that will run the `proxy`.

### Running p2p Backend

```bash
$ ./p2p/seawolf_p2p
```