FROM golang:1.26-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o main ./cmd/main.go

FROM alpine:latest

WORKDIR /root/

RUN apk --no-cache add ca-certificates

COPY certs/russian_trusted_root_ca.crt /usr/local/share/ca-certificates/russian_trusted_root_ca.crt
COPY certs/russian_trusted_sub_ca.crt /usr/local/share/ca-certificates/russian_trusted_sub_ca.crt
RUN update-ca-certificates


COPY --from=builder /app/main .
COPY --from=builder /app/static ./static

RUN chmod +x ./main

CMD ["./main"]

