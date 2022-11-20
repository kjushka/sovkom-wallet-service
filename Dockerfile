FROM golang:1.18-alpine
WORKDIR /wallet-service
COPY / ./
RUN go mod download
RUN go build -o ./wallet-service ./cmd/wallet-service
ENTRYPOINT [ "./wallet-service" ]