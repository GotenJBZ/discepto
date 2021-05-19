# Discepto
A debate/discussion platform.

## Build instructions
### Dependencies
You need
- go
- postgresql
- make (to build the release binary)

```bash
git clone $repo_url
cd $repo_url

# To run:
# When developing locally, set DEBUG to true.
# Pass the correct database creds
export DISCEPTO_DATABASE_URL="postgres://postgres:passwdpost@localhost/discepto?sslmode=disable"
go run cmd/discepto/main.go start

# To build release:
make release
```

### Environment variables
```
DISCEPTO_DEBUG: when `true`, always reload html templates, improve debugging/logging
DISCEPTO_DATABASE_URL: for example "postgres://user:password@localhost/database"
DISCEPTO_PORT: Discepto will open and listen on this TCP port. Default is 23495
DISCEPTO_SESSION_KEY: Key used to sign cookies
```
