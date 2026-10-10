FROM golang:alpine AS builder

WORKDIR /app

COPY . .

RUN go mod tidy && CGO_ENABLED=0 GOOS=linux go build -o server main.go

FROM alpine:latest

WORKDIR /app

COPY --from=builder /app/server .

EXPOSE 8081

CMD ["./server"]