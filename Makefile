.DEFAULT_GOAL := build

##################################
# git
##################################
GIT_URL ?= $(shell git remote get-url --push origin)
GIT_COMMIT ?= $(shell git rev-parse HEAD)
#GIT_SHORT_COMMIT := $(shell git rev-parse --short HEAD)
TIMESTAMP := $(shell date '+%Y-%m-%d_%I:%M:%S%p')
REGION ?= us-east-2
IMAGE_REGISTRY ?= <REGISTRY>
IMAGE_REPO ?= <REPO>
DOCKERFILE ?= Dockerfile
NO_CACHE ?= true
GIT_COMMIT_IN ?=
GIT_URL_IN ?=
GO_MOD_PATH ?= jimmyray.io/data-api/main

ifeq ($(strip $(GIT_COMMIT)),)
GIT_COMMIT := $(GIT_COMMIT_IN)
endif

ifeq ($(strip $(GIT_URL)),)
GIT_URL := $(GIT_URL_IN)
endif

VERSION_HASH := $(shell echo $(GIT_COMMIT)|cut -c 1-10)
# $(info [$(VERSION_HASH)])
VERSION_FROM_FILE ?= $(shell head -n 1 version)
VERSION ?=

ifeq ($(strip $(VERSION_HASH)),)
VERSION := $(VERSION_FROM_FILE)
else
VERSION := $(VERSION_FROM_FILE)-$(VERSION_HASH)
endif

.PHONY: build push pull meta clean compile init check

build:	meta
	$(info    [BUILD_CONTAINER_IMAGE])
	docker build -f $(DOCKERFILE) . -t $(IMAGE_REGISTRY)/$(IMAGE_REPO):$(VERSION) --no-cache=$(NO_CACHE)
	$(info	)

login:
	aws ecr get-login-password --region $(REGION) | docker login --username AWS --password-stdin $(IMAGE_REGISTRY)

push:	meta
	$(info    [PUSH_CONTAINER_IMAGE])
	docker push $(IMAGE_REGISTRY)/$(IMAGE_REPO):$(VERSION)
	$(info	)

pull:	meta
	$(info    [PULL_CONTAINER_IMAGE])
	docker pull $(IMAGE_REGISTRY)/$(IMAGE_REPO):$(VERSION)
	$(info	)

meta:
	$(info    [METADATA])
	$(info    timestamp: [$(TIMESTAMP)])
	$(info    git commit: [$(GIT_COMMIT)])
	$(info    git URL: [$(GIT_URL)])
	$(info    Container image version: [$(VERSION)])
	$(info	)

compile:	clean	meta
	$(info   [COMPILE])
	go env -w GOPROXY=direct && CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -a -installsuffix cgo -o main.bin .
	$(info	)

clean:
	-@rm main.bin

init:
	-@rm go.mod
	-@rm go.sum
	go mod init $(GO_MOD_PATH)
	go mod tidy

check:
	-go vet main
	-golangci-lint run

