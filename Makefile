# moodns Makefile
# Copyright (C) 2014 Alessandro Ghedini <alessandro@ghedini.me>
# This file is released under the 2 clause BSD license, see COPYING

export GOPATH:=$(GOPATH):$(CURDIR)

BUILDTAGS=debug

all: moodns

moodns: deps
	go install -tags '$(BUILDTAGS)' main/moodns
	go install -tags '$(BUILDTAGS)' main/moodns-resolve

deps:
	go get -tags '$(BUILDTAGS)' -d -v main/moodns
	go get -tags '$(BUILDTAGS)' -d -v main/moodns-resolve

man: docs/moodns.1.md docs/moodns-resolve.1.md
	ronn -r $?

html: docs/moodns.1.md docs/moodns-resolve.1.md
	ronn -h $?

release-all: BUILDTAGS=release
release-all: all

clean:
	go clean -i main/moodns main/moodns-resolve mdns

.PHONY: all moodns deps clean
