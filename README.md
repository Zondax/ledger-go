# ledger-goclient

This project is work in progress. Some aspects are subject to change.

# Get source
Apart from cloning, be sure you install dep dependency management tool
https://github.com/golang/dep

## Setup
Update dependencies using the following:
```
dep ensure 
```

# Building
```
go build ledger.go
```

# Running
./ledger

Make sure that the app is launched in the Ledger before starting this command and Ledger is connected to the USB port.
This command line tool will try to send a simple json transaction and will return a signature when user agrees to sign.
