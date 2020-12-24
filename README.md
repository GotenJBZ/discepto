# Discepto
A debate/discussion platform.

## Build instructions
### Dependencies
You need
- go
- postgresql
- make
- pkger (`go get github.com/markbates/pkger/cmd/pkger`) 

```bash
git clone $repo_url
cd $repo_url

# To run:
# When developing locally, set DEBUG to true.
# Pass the correct database creds
DEBUG=true DATABASE_URL="postgres://user:passwd@localhost/disceptoDb" make run

# To build release:
make release
```

### Environment variables
DEBUG: when `true`, reload html templates every request
DATABASE_URL: you know
PORT: you know. Default is 23495
