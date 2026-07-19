PREFIX ?= /usr/local
BINDIR ?= $(PREFIX)/bin
INSTALL_GOROOT ?= $(PREFIX)/lib/go
DESTDIR ?=

GOROOT_DIRS := api bin doc lib misc src

.PHONY: build install

build:
	cd src && ./make.bash

install: build
	test -x bin/go
	test -x bin/gofmt
	test -f VERSION.cache
	set -eu; \
	target="$(DESTDIR)$(INSTALL_GOROOT)"; \
	parent=$$(dirname "$$target"); \
	install -d "$$parent"; \
	stage=$$(mktemp -d "$$parent/.go-install.XXXXXX"); \
	backup=; \
	cleanup() { \
		status=$$?; \
		trap - EXIT HUP INT TERM; \
		if test -n "$$backup" && test ! -e "$$target" && test ! -L "$$target"; then \
			mv -- "$$backup" "$$target" || status=$$?; \
		fi; \
		test -z "$$stage" || rm -rf -- "$$stage" || status=$$?; \
		exit "$$status"; \
	}; \
	trap cleanup EXIT; \
	trap 'exit 129' HUP; \
	trap 'exit 130' INT; \
	trap 'exit 143' TERM; \
	cp -R $(GOROOT_DIRS) "$$stage/"; \
	install -d "$$stage/pkg"; \
	cp -R pkg/include pkg/tool "$$stage/pkg/"; \
	install -m 0644 go.env "$$stage/go.env"; \
	install -m 0644 VERSION.cache "$$stage/VERSION"; \
	if test -e "$$target" || test -L "$$target"; then \
		backup="$$target.backup.$$$$"; \
		test ! -e "$$backup"; \
		mv -- "$$target" "$$backup"; \
	fi; \
	if ! mv -- "$$stage" "$$target"; then \
		test -z "$$backup" || mv -- "$$backup" "$$target"; \
		exit 1; \
	fi; \
	stage=; \
	test -z "$$backup" || rm -rf -- "$$backup"
	install -d "$(DESTDIR)$(BINDIR)"
	ln -sfn "$(INSTALL_GOROOT)/bin/go" "$(DESTDIR)$(BINDIR)/go"
	ln -sfn "$(INSTALL_GOROOT)/bin/gofmt" "$(DESTDIR)$(BINDIR)/gofmt"
