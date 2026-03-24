FROM golang:1.25.7-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN GOOS=linux go build -o tabula .

FROM alpine:latest

WORKDIR /app
COPY --from=builder /app/tabula .

CMD ["./tabula"]