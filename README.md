# Discepto
A debate/discussion platform.

## Build instructions
### Dependencies
You need
- go
- postgresql

```bash
git clone $repo_url
cd $repo_url

# To run:
# When developing locally, set DEBUG to true.
# Pass the correct database creds
DEBUG=true DATABASE_URL="postgres://user:passwd@localhost/disceptoDb" go run cmd/discepto/main.go

# To build release:
go build main.go
```
