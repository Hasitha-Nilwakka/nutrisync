FROM golang:1.24-alpine AS builder

WORKDIR /app

COPY go.mod ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /go-api .

FROM alpine:3.21

RUN adduser -D -g '' appuser
WORKDIR /app

COPY --from=builder /go-api /app/go-api

USER appuser
EXPOSE 8080

CMD ["/app/go-api"]
