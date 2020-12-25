release:
	mkdir -p build/discepto/
	cp -r web/ migrations/ build/discepto
	CGO_ENABLED=0 go build -ldflags "-s -w -extldflags '-static'" -o build/discepto/discepto gitlab.com/ranfdev/discepto/cmd/discepto/

pack: release
	tar -czf build/discepto.tar.gz -C build/ discepto/ --remove-files

run:
	go run cmd/discepto/main.go

clean:
	rm -rf build/

.PHONY: run release pack clean
