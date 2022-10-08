APP := mastopurge
VERSION := $(git describe --tags --exact-match || git rev-parse --short HEAD)

# -s -w: omit debug and similar symbols.
LD_FLAGS := "-s -w -X 'main.versionString=$(VERSION)'"

PKG_LIST := mastopurge.go api.go

.PHONY: clean \
	test \
	run \
	lint

mastopurge: build

# Starts at generate.go. Also processes all occurrances of //go:generate
generate:
	@ go generate

build: $(PKG_LIST)
	@ go build -o $(APP) -ldflags=$(LD_FLAGS) $(PKG_LIST)

# Debug build
debug: $(PKG_LIST)
	@ go build -o $(APP) $(PKG_LIST)

run:
	@ go run $(PKG_LIST)

clean:
	@ go clean -modcache
	@ rm -f $(APP)

test:
	go test -cover -race -count=1 ./...
