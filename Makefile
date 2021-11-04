#!/usr/bin/make

.DEFAULT_GOAL := all
PLATFORMS := linux/amd64 darwin/amd64 windows/amd64

temp = $(subst /, ,$@)
os = $(word 1, $(temp))
arch = $(word 2, $(temp))

.PHONY: setup
setup:
	@go mod download

.PHONY: bin
bin:
	go build -o ./dist/vault-backup

.PHONY: releases
releases: $(PLATFORMS)

$(PLATFORMS):
	GOOS=$(os) GOARCH=$(arch) go build -o 'dist/vault-backup_$(os)-$(arch)'

.PHONY: all
all:
	@make -s bin

