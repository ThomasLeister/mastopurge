#!/bin/bash

### Compile and link statically
CGO_ENABLED=0 GOOS=linux go build -a -ldflags '-extldflags "-static" -w -s' mastopurge.go api.go
