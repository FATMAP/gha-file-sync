#!make

include .local-testing-env
.EXPORT_ALL_VARIABLES:

MAKEFLAGS += \
--warn-undefined-variables
SHELL = /usr/bin/env bash
.SHELLFLAGS := -O globstar -O extglob -eu -o pipefail -c

.PHONY: run
run:
	go build &&	./git-file-sync
