# Discepto
A debate/discussion platform.

## Build instructions
### Dependencies
You need
- go
- postgresql
- make

```bash
git clone $repo_url
cd $repo_url

# To run:
# When developing locally, set DEBUG to true.
# Pass the correct database creds
DISCEPTO_DATABASE_URL="postgres://user:passwd@localhost/disceptoDb" make run

# To build release:
make release
```

### Environment variables
```
DISCEPTO_DEBUG: when `true`, reload html templates every request
DISCEPTO_DATABASE_URL: you know
DISCEPTO_PORT: you know. Default is 23495
DISCEPTO_SESSION_KEY: key used to encrypt cookies
```
