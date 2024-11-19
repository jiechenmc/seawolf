FROM golang:1.23

WORKDIR /app

COPY ./ ./

RUN make

CMD [ "./seawolf_p2p" ]

EXPOSE 8081