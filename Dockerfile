FROM golang:1.19-alpine
WORKDIR /wallet-service
COPY / ./
RUN go mod download
RUN go build -o ./bin/wallet-service ./cmd/wallet-service

CMD [ "./bin/wallet-service" ]