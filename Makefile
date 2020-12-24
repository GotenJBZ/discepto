release:
	pkger
	go build -race -ldflags "-s -w -extldflags '-static'" -o discepto cmd/discepto/main.go


run:
	go run cmd/discepto/main.go
