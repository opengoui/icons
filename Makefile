# Lucide release to sync: a tag such as 1.53.0, "latest", or empty to keep the
# version recorded in index.json.
VERSION ?=
# Optional local Lucide source tarball for offline syncs (requires VERSION).
ARCHIVE ?=

SYNC_FLAGS := $(if $(VERSION),-version $(VERSION)) $(if $(ARCHIVE),-archive $(ARCHIVE))

.PHONY: help sync upgrade test check

help:
	@echo "make sync [VERSION=1.53.0] [ARCHIVE=lucide.tgz]  re-sync svg/, index.json, name/ and LICENSE.lucide"
	@echo "make upgrade                                     sync the latest Lucide release"
	@echo "make test                                        run all tests"
	@echo "make check                                       vet, race tests, tidy and cross-build"

sync:
	go run ./internal/sync $(SYNC_FLAGS)

upgrade:
	go run ./internal/sync -version latest

test:
	go test ./... -count=1

check:
	go vet ./...
	go test -race ./... -count=1
	go mod tidy -diff
	GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build ./...
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build ./...
	GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 go build ./...
