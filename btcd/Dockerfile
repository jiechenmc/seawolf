FROM golang:1.23


WORKDIR /app 

RUN git clone https://github.com/btcsuite/btcd.git

WORKDIR /app/btcd

RUN go install -v . ./cmd/...


CMD [ "btcd", "-u", "rpcuser", "-P", "rpcpass", "--simnet" ]

ENV PATH="/go/bin:${PATH}"
