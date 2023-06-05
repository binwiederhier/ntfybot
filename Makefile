MAKEFLAGS := --jobs=1

.PHONY:

help:
	@echo "Build:"
	@echo "  make build                      - Build ntfybot"
	@echo "  make clean                      - Clean build/dist folders"
	@echo
	@echo "Releasing:"
	@echo "  make release                    - Create a release"
	@echo "  make release-snapshot           - Create a test release"


# Building everything

clean: .PHONY
	rm -rf dist build

build-deps: .PHONY
	go install github.com/goreleaser/goreleaser@latest

build: build-deps
	goreleaser build --snapshot --clean

release: clean build-deps
	goreleaser release --clean

release-snapshot: clean build-deps
	goreleaser release --snapshot --skip-publish --clean

