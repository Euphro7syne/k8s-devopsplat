FROM golang:1.22-alpine AS build

ARG GOPROXY=https://goproxy.cn,direct

WORKDIR /src
RUN apk add --no-cache build-base
COPY go.mod go.sum ./
RUN GOPROXY="${GOPROXY}" go mod download
COPY . .
RUN GOPROXY="${GOPROXY}" go build -o /out/ops-server ./cmd/ops-server

FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata
WORKDIR /app
COPY --from=build /out/ops-server /usr/local/bin/ops-server
COPY configs/ops-server.example.yaml /etc/ops-platform/ops-server.yaml
ENTRYPOINT ["ops-server"]
CMD ["-config", "/etc/ops-platform/ops-server.yaml"]
