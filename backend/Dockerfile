FROM golang:1.26-alpine AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 go build -o /out/migrate ./cmd/migrate \
    && CGO_ENABLED=0 go build -o /out/api ./cmd/api

FROM alpine:3.22

RUN apk add --no-cache ca-certificates

COPY --from=builder /out/migrate /usr/local/bin/migrate
COPY --from=builder /out/api /usr/local/bin/api

EXPOSE 8080

CMD ["sh", "-c", "migrate && exec api"]