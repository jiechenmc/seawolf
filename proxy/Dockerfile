FROM golang:1.23

WORKDIR /app

COPY ./ ./

RUN go build

CMD [ "./proxy" ]

EXPOSE 8082