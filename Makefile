.PHONY: build

default: build

build:
	go build -i -o ./blackbox_exporter .
