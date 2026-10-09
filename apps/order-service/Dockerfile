FROM golang:1.26-alpine AS build

WORKDIR /workspace

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -o /out/order-service ./cmd/server

FROM alpine:3.20

RUN apk add --no-cache ca-certificates

WORKDIR /app

RUN addgroup -S order && adduser -S order -G order

COPY --from=build /out/order-service ./order-service

USER order

EXPOSE 8084

ENTRYPOINT ["./order-service"]
