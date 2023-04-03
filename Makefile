#!make

include .local-testing-env
.EXPORT_ALL_VARIABLES:

MAKEFLAGS += \
--warn-undefined-variables
SHELL = /usr/bin/env bash
.SHELLFLAGS := -O globstar -O extglob -eu -o pipefail -c

.PHONY: run
run:
	go build &&	./github-file-sync

.PHONY: lint
lint:
	golangci-lint run --config .golangci.yml --timeout 3m
