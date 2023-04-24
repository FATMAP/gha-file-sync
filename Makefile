#!make

MAKEFLAGS += \
--warn-undefined-variables
SHELL = /usr/bin/env bash
.SHELLFLAGS := -O globstar -O extglob -eu -o pipefail -c

.PHONY: lint
lint:
	golangci-lint run --config .golangci.yml --timeout 3m
