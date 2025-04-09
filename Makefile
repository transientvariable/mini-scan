SHELL := /bin/bash

# ================================================
# Config
# ================================================

GO:=$(GOROOT)/bin/go
export PATH:=$(GOROOT)/bin:$(PATH)

ENV               ?= dev
PROJECT_NAME      := transientvariable/mini-scan
COMMIT            := $(shell git rev-parse --short HEAD)
BIN_NAME          := scanner
BUILD_TIMESTAMP   := $(date -u +'%Y-%m-%dT%H:%M:%SZ')
BUILD_OUTPUT_DIR  := build
DEPLOY_DIR        := deploy
DOCKERFILES       := $(DEPLOY_DIR)/docker

MAKEFLAGS += --warn-undefined-variables
MAKEFLAGS += --no-builtin-rules
MAKEFLAGS += --silent

# ================================================n
# Rules
# ================================================

default: all

.PHONY: all
all: clean check build build.docker

.PHONY: clean
clean:
	@printf "\033[2m→ Cleaning project build output directory: $(BUILD_OUTPUT_DIR)\033[0m\n"
	@rm -rf "$(BUILD_OUTPUT_DIR)" 2> /dev/null

.PHONY: check
check:
	@printf "\033[2m→ No checks for this repository at this time...\033[0m\n"

.PHONY: build.all
build.all: clean build

.PHONY: build
build:
	@printf "\033[2m→ Building application binary...\033[0m\n"
	@mkdir -p $(BUILD_OUTPUT_DIR)
	@go mod download && go mod verify
	@go build -ldflags="-w -s" -o $(BUILD_OUTPUT_DIR)/$(BIN_NAME) .

# ================================================
# Rules : Docker
# ================================================

.PHONY: build.docker
build.docker:
	@printf "\033[2m→ Docker build...\033[0m\n"
	DOCKER_BUILDKIT=1 docker image build \
        --build-arg ENV=$(ENV) \
		--build-arg BUILD_DATE="$(BUILD_TIMESTAMP)" \
		-t $(PROJECT_NAME) -f $(DOCKERFILES)/Dockerfile .
