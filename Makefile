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
	install -d "$(DESTDIR)$(INSTALL_GOROOT)"
	cp -R $(GOROOT_DIRS) "$(DESTDIR)$(INSTALL_GOROOT)/"
	install -d "$(DESTDIR)$(INSTALL_GOROOT)/pkg"
	cp -R pkg/include pkg/tool "$(DESTDIR)$(INSTALL_GOROOT)/pkg/"
	install -m 0644 go.env "$(DESTDIR)$(INSTALL_GOROOT)/go.env"
	install -m 0644 VERSION.cache "$(DESTDIR)$(INSTALL_GOROOT)/VERSION"
	install -d "$(DESTDIR)$(BINDIR)"
	ln -sfn "$(INSTALL_GOROOT)/bin/go" "$(DESTDIR)$(BINDIR)/go"
	ln -sfn "$(INSTALL_GOROOT)/bin/gofmt" "$(DESTDIR)$(BINDIR)/gofmt"
