FROM golang:1.23


WORKDIR /app 

RUN apt update
RUN git clone https://github.com/prithesh07/btcd.git
RUN git clone https://github.com/tahsina13/btcwallet

# Install btcd
WORKDIR /app/btcd
RUN go install -v . ./cmd/...

# Install the wallet
WORKDIR /app/btcwallet
RUN go install -v . ./cmd/...

ENV PATH="/go/bin:${PATH}"
ENV btcdusername="rpcuser"
ENV btcdpassword="rpcpass"
# ENV btcdnetwork="mainnet"

RUN echo "alias bctl='btcctl --notls --rpcuser=$btcdusername --rpcpass=$btcdpassword'" >> /root/.bashrc

WORKDIR /app/server

EXPOSE 8080

CMD [ "go", "run", "main.go" ]