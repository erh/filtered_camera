
UNAME := $(shell uname -s)

FLAGS=-tags no_cgo,osusergo,netgo

ifeq ($(UNAME),Linux)
    FLAGS += -ldflags="-extldflags=-static -s -w"
endif

filtered-camera: Makefile *.go cmd/module/*.go
	go build $(FLAGS) -o filtered-camera cmd/module/cmd.go

test:
	go test

lint:
	gofmt -w -s .

module: filtered-camera
	tar czf module.tar.gz filtered-camera

all: module test

update:
	go get go.viam.com/rdk@latest
	go mod tidy
