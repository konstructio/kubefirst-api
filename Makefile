XC_OS="linux darwin"
XC_ARCH="amd64 arm64"
XC_PARALLEL="2"
BIN="bin/"
SRC=$(shell find . -name "*.go")

ifeq (, $(shell which gox))
$(warning "could not find gox in $(PATH), run: go install github.com/mitchellh/gox@latest")
endif

.PHONY: all build

default: all

all: build

build:
	gox \
		-os=$(XC_OS) \
		-arch=$(XC_ARCH) \
		-parallel=$(XC_PARALLEL) \
		-output=$(BIN)/{{.Dir}}_{{.OS}}_{{.Arch}} \
		;
