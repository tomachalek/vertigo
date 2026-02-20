.PHONY: build test clean

build:
	go build -o benchmark ./cmd/benchmark/

test:
	go test ./...

clean:
	rm -rf benchmark
