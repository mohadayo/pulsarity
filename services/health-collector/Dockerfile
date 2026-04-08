FROM golang:1.22-alpine AS builder

WORKDIR /app
COPY go.mod ./
COPY *.go ./
RUN go build -o health-collector .

FROM alpine:3.20
WORKDIR /app
COPY --from=builder /app/health-collector .

EXPOSE 8002

CMD ["./health-collector"]
