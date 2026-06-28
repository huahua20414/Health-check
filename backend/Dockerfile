FROM golang:1.25-alpine AS builder

ENV GOPROXY=https://goproxy.cn,direct
ENV CGO_ENABLED=0
ENV GOMAXPROCS=1
ENV GOFLAGS=-p=1
ENV GOGC=50

WORKDIR /src
COPY go.mod go.sum* ./
RUN go mod download
COPY . .
RUN go build -ldflags="-s -w" -o /out/server ./cmd/server

FROM alpine:3.21

WORKDIR /app
COPY --from=builder /out/server /app/server
EXPOSE 8080
CMD ["/app/server", "serve"]
