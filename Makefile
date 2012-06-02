# Requires gzip, txt2man
# TODO add txt2man doc generation

DESTDIR     ?= ""
PREFIX      ?= $(DESTDIR)/usr
MANPREFIX   ?= $(PREFIX)/share/man

# install paths
INSTALL_BIN       = $(PREFIX)/bin
INSTALL_MAN1      = $(MANPREFIX)/man1
INSTALL_SERVICE   = $(PREFIX)/lib/systemd/system
INSTALL_ETC       = $(DESTDIR)/etc
INSTALL_RCD       = $(INSTALL_ETC)/rc.d
INSTALL_CRONJOB   = $(INSTALL_ETC)/cron.hourly
INSTALL_TMPFILESD = $(PREFIX)/lib/tmpfiles.d
SYSCONFDIR        = $(PREFIX)/etc

SRCDIR      := src
DOCDIR      := doc
BUILDDIR    := build
BIN         := goanysync

SRC_POSTFIX := .go
DOC_POSTFIX := .man.txt
MAN_POSTFIX := .1.gz
SOURCES     := $(wildcard $(SRCDIR)/*$(SRC_POSTFIX))
DOCS        := $(wildcard $(DOCDIR)/*$(DOC_POSTFIX))

# Man page targets
MANPAGES := $(DOCS:$(DOCDIR)/%=$(BUILDDIR)/%)
MANPAGES := $(MANPAGES:%$(DOC_POSTFIX)=%$(MAN_POSTFIX))

# Create the build dir
ifeq ($(wildcard $(BUILDDIR)/),)
	_       := $(shell mkdir -p $(BUILDDIR))
endif

##############################################

.PHONY: all
all: $(BUILDDIR)/$(BIN) $(MANPAGES)


$(BUILDDIR)/$(BIN): $(SOURCES)
	@go build -o $(BUILDDIR)/$(BIN) $(SOURCES)


$(BUILDDIR)/%$(MAN_POSTFIX): $(DOCDIR)/%$(DOC_POSTFIX)
	@txt2man $(DOCDIR)/$*$(DOC_POSTFIX) | gzip -9 -c > $(BUILDDIR)/$*$(MAN_POSTFIX)


.PHONY: clean
clean:
	@rm -f $(BUILDDIR)/$(BIN)
	@rm -f $(BUILDDIR)/*.1.gz
	@if [[ "${BUILDDIR}" != "." && "${BUILDDIR}" != "./" ]]; then rmdir -- $(BUILDDIR); fi;


.PHONY: install
install: all
	@[[ -f /etc/arch-release ]] && \
		install -D --mode=0755 script/gsd      "$(INSTALL_RCD)/gsd"
	@install -D --mode=0744 conf/gsd.cronjob   "$(INSTALL_CRONJOB)/gsd"
	@install -D --mode=0644 conf/tmpfiles.conf "$(INSTALL_TMPFILESD)/goanysync.conf"
	@mkdir -p --mode=0755 "$(INSTALL_SERVICE)" "$(INSTALL_BIN)" "$(INSTALL_ETC)" "$(INSTALL_MAN1)"
	@install --mode=0644 --target-directory="$(INSTALL_SERVICE)" conf/goanysync.service
	@install --mode=0755 --target-directory="$(INSTALL_BIN)" "$(BUILDDIR)/$(BIN)"
	@install --mode=0644 --target-directory="$(INSTALL_ETC)" conf/goanysync.conf
	@install --mode=0644 --target-directory="$(INSTALL_MAN1)" $(MANPAGES)
