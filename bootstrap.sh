sudo docker compose up --build -d
sudo docker cp seawolf-btcd-1:/root/.btcd/rpc.cert ./btcd.cert
sudo docker cp btcd.cert seawolf-btcwallet-1:/root/.btcwallet/
sudo rm btcd.cert