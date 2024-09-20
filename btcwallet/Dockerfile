FROM golang:1.23

WORKDIR /app 

RUN git clone https://github.com/btcsuite/btcwallet.git

WORKDIR /app/btcwallet

RUN go install -v . ./cmd/...

# RUN btcwallet -u rpcuser -P rpcpass --create 


ENV PATH="/go/bin:${PATH}"
ENV btcdpassword="rpcpass"
ENV btcdusername="rpcuser"

# RUN btcwallet -u rpcuser -P rpcpass
