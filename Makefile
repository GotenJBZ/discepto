release:
	mkdir -p build/discepto/
	cp -r web/ migrations/ build/discepto
	CGO_ENABLED=0 go build -ldflags "-s -w -extldflags '-static'" -o build/discepto/discepto gitlab.com/ranfdev/discepto/cmd/discepto/

pack: release
	tar -czf build/discepto.tar.gz -C build/ discepto/ --remove-files

build:
	mkdir -p build/discepto/
	go build -o build/discepto/discepto gitlab.com/ranfdev/discepto/cmd/discepto/

run: build
	DISCEPTO_DEBUG=true ./build/discepto/discepto start

test:
	go fmt `go list ./... | grep -v /vendor/`
	go vet `go list ./... | grep -v /vendor/`
	go test -race `go list ./... | grep -v /vendor/`
clean:
	rm -rf build/

.PHONY: run release pack clean test build
