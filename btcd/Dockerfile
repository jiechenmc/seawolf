FROM golang:1.23


WORKDIR /app 

RUN apt update && apt install tmux -y
RUN git clone https://github.com/btcsuite/btcd.git
RUN git clone https://github.com/btcsuite/btcwallet.git

# Install btcd
WORKDIR /app/btcd
RUN go install -v . ./cmd/...

# Install the wallet
WORKDIR /app/btcwallet
RUN go install -v . ./cmd/...

ENV PATH="/go/bin:${PATH}"
ENV btcdusername="rpcuser"
ENV btcdpassword="rpcpass"
ENV btcdnetwork="simnet"

RUN echo "alias bctl='btcctl --$btcdnetwork --rpcuser=$btcdusername --rpcpass=$btcdpassword'" >> /root/.bashrc

# CMD [ "btcd", "-u", "rpcuser", "-P", "rpcpass", "--simnet", "--miningaddr", "SZoGnna9NsjkZWusgFJ3DGirJpq22GqmES" ]