#!/usr/bin/make -f
# -*- makefile -*-

P = $(shell basename $(CURDIR))
M = $(shell printf "\033[34;1m▶\033[0m")

# default config for the build step
CGO_ENABLED ?= 0
GOOS ?= linux
GOARCH ?= amd64

BUILD_NUMBER ?= local
COMMIT ?= $(shell git rev-parse --short HEAD)$(shell test -n "`git status --porcelain`" && echo "+CHANGES" || true)

.PHONY: help
help:
	@echo 'Management commands for $(P):'
	@echo
	@echo 'Usage:'
	@echo '    make build           Compile the project.'
	@echo '    make revendor        Update go modules and tidy vendor directory.'
	@echo '    make test            Vet and test project.'
	@echo
	@echo '    make up              Start services defined in docker-compose.yml.'
	@echo '    make reup            Rebuild and recreate services defined in docker-compose.yml.'
	@echo '    make down            Stop and cleanup services defined in docker-compose.yml.'
	@echo '    make logs            Tail docker logs of services defined in docker-compose.yml.'
	@echo

build: ; $(info $(M) Running build...)
	CGO_ENABLED=$(CGO_ENABLED) GOOS=$(GOOS) GOARCH=$(GOARCH) \
	go build -mod=vendor \
	-ldflags '-w -extldflags "-static" -X main.Commit=$(COMMIT) -X main.Build=$(BUILD_NUMBER)' \
	-o release/linux/amd64/prometheus-es-adapter cmd/adapter/main.go

.PHONY: revendor
revendor: ; $(info $(M) Updating vendor dependencies...)
	go get -u
	go mod tidy
	go mod vendor

.PHONY: test
test: ; $(info $(M) Running tests...)
	go vet -mod=vendor ./...
	go test -mod=vendor -cover -coverprofile=coverage.out ./...

.PHONY: race
race:
	go test -mod=vendor -race ./...

########## docker and docker-compose related commands follow ##########

.PHONY: up
up:
	docker-compose -f docker-compose.yml up -d

.PHONY: reup
reup: build
	docker-compose -f docker-compose.yml up --build -d

.PHONY: down
down:
	docker-compose -f docker-compose.yml down

.PHONY: logs
logs:
	docker-compose -f docker-compose.yml logs -f
