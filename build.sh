#!/bin/bash

### Get VERSIONSTRING from Git
VERSIONSTRING="$(git describe --tags --exact-match || git rev-parse --short HEAD)"

### Compile and link statically
echo "Building version ${VERSIONSTRING} of MastoPurge ..."
CGO_ENABLED=0 GOOS=linux go build -a -ldflags "-extldflags '-static' -w -s -X main.versionString=${VERSIONSTRING}" mastopurge.go api.go
