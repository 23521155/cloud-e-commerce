FROM golang:1.26-alpine AS build

WORKDIR /workspace

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -o /out/catalog-service ./cmd/server
RUN CGO_ENABLED=0 go build -o /out/seed-job ./cmd/seed

FROM alpine:3.20

RUN apk add --no-cache ca-certificates

WORKDIR /app

RUN addgroup -S catalog && adduser -S catalog -G catalog

COPY --from=build /out/catalog-service ./catalog-service
# One-off job run by compose (catalog-seed); not part of the service itself.
COPY --from=build /out/seed-job ./seed-job
COPY seed/books.jsonl.gz ./seed/books.jsonl.gz

USER catalog

EXPOSE 8083

ENTRYPOINT ["./catalog-service"]
