.PHONY: all build build-amd64 build-arm64 force-build clean help

CGO_CFLAGS=-mmacosx-version-min=12.3

CODESIGN_IDENTITY ?= -

all: help

##@
##@ Build commands
##@

build: ##@ Build binaries for all architectures
	@$(MAKE) out/ovm-amd64
	@$(MAKE) out/ovm-arm64

build-amd64: ##@ Build amd64 binary
	@$(MAKE) out/ovm-amd64

build-arm64: ##@ Build arm64 binary
	@$(MAKE) out/ovm-arm64

out/ovm-arm64 out/ovm-amd64: out/ovm-%: force-build
	@mkdir -p $(@D)
	CGO_ENABLED=1 CGO_CFLAGS=$(CGO_CFLAGS) GOOS=darwin GOARCH=$* go build -o $@ ./cmd/ovm
	codesign --force --options runtime --entitlements ovm.entitlements --sign $(CODESIGN_IDENTITY) $@

force-build:


##@
##@ Clean commands
##@

clean: ##@ Clean up build artifacts
	$(RM) -rf out


##@
##@ Misc commands
##@

help: ##@ (Default) Print listing of key targets with their descriptions
	@printf "\nUsage: make <command>\n"
	@grep -F -h "##@" $(MAKEFILE_LIST) | grep -F -v grep -F | sed -e 's/\\$$//' | awk 'BEGIN {FS = ":*[[:space:]]*##@[[:space:]]*"}; \
	{ \
		if($$2 == "") \
			printf ""; \
		else if($$0 ~ /^#/) \
			printf "\n%s\n", $$2; \
		else if($$1 == "") \
			printf "     %-20s%s\n", "", $$2; \
		else \
			printf "\n    \033[34m%-20s\033[0m %s\n", $$1, $$2; \
	}'
