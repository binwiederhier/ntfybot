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

build: .PHONY
	goreleaser build --snapshot --clean

release: clean
	goreleaser release --clean

release-snapshot: clean
	goreleaser release --snapshot --skip-publish --clean

