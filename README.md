# zondaX-ledger

This project is work in progress. Some aspects are subject to change.

# Get source
Apart from cloning, be sure you install dep dependency management tool
https://github.com/golang/dep

## Ubuntu
Update dependencies using the following:
```
dep ensure -update
```

# Building
```
go build ledger.go
```

# Running
./ledger

It should print nothing if ledger is correctly detected or error otherwise.
