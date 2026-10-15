FROM golang:1.26-alpine AS build

WORKDIR /workspace

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -o /out/user-service ./cmd/server

FROM alpine:3.20

RUN apk add --no-cache ca-certificates

WORKDIR /app

RUN addgroup -S account && adduser -S account -G account

COPY --from=build /out/user-service ./user-service

USER account

EXPOSE 8085

ENTRYPOINT ["./user-service"]
