.PHONY: build test clean deploy

# Build the Wasm binary with TinyGo
build:
	tinygo build -target=wasi -gc=conservative -o bin/main.wasm .

# Run tests (skip Fastly host calls)
test:
	go test -tags nofastlyhostcalls ./...

# Clean build artifacts
clean:
	rm -f bin/main.wasm

# Deploy to Fastly Compute@Edge
deploy:
	fastly compute publish
