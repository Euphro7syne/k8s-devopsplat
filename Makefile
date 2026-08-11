SHELL := /bin/bash

CONFIG ?= configs/ops-server.example.yaml
SERVER_IMAGE ?= ops-platform/ops-server:latest
WEB_IMAGE ?= ops-platform/ops-web:latest
GOPROXY ?= https://goproxy.cn,direct
GO_PACKAGES ?= ./cmd/... ./internal/...
export GOCACHE ?= $(CURDIR)/.cache/go-build
export GOMODCACHE ?= $(CURDIR)/.cache/go-mod
export npm_config_cache ?= $(CURDIR)/.cache/npm
export GOPROXY

.PHONY: deps run-server migrate lint test build image deploy backup restore

deps:
	go mod download
	cd web && npm install

run-server:
	go run ./cmd/ops-server -config $(CONFIG)

migrate:
	go run ./cmd/ops-migrate -config $(CONFIG)

lint:
	test -z "$$(gofmt -l $$(find cmd internal -name '*.go'))"
	go vet $(GO_PACKAGES)
	cd web && npm run lint

test:
	go test $(GO_PACKAGES)
	cd web && npm run test

build:
	mkdir -p bin
	go build -o bin/ops-server ./cmd/ops-server
	go build -o bin/ops-migrate ./cmd/ops-migrate
	cd web && npm run build

image:
	docker build -f deploy/docker/ops-server.Dockerfile -t $(SERVER_IMAGE) .
	docker build -f deploy/docker/ops-web.Dockerfile -t $(WEB_IMAGE) web

deploy:
	kubectl apply -f deploy/k3s

backup:
	./scripts/backup.sh

restore:
	./scripts/restore.sh
