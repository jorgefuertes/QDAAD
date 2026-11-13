# La Aventura Original for a Spectrum +3, with the 16-bit editions' texts.
#
# Run from this directory. The tools are run out of the repository rather than
# taken from the PATH, so a fresh checkout needs nothing installed:
#
#   make dist    both disks, in dist/
#   make test    both walkthroughs played, built in tmp/
#   make clean   neither directory left behind
#
# QDAAD, QPACK and QRUN are overridable, so an installed build is one
# assignment away:  make dist QDAAD=qdaad QPACK=qpack

.PHONY: dist test clean

ROOT  := ../../../..
QDAAD ?= go run $(ROOT)/cmd/qdaad
QPACK ?= go run $(ROOT)/cmd/qpack
QRUN  ?= go run $(ROOT)/cmd/qrun

# The tests are shared with the other trees and are not copies: these are the
# -extended ones, which are the same walkthroughs quoting the long texts.
TESTS := ../../test

dist:
	@mkdir -p tmp dist
	@for part in 1 2; do \
		$(QDAAD) compile -i part$$part/game.sce -o tmp/build-part$$part \
			-f build -m zx -s plus3 -v 3 --part $$part || exit 1; \
		$(QPACK) -f dsk -o dist/AO$$part --name AO$$part tmp/build-part$$part || exit 1; \
	done

# Built for the PC rather than the Spectrum because qrun plays a database and
# not a disk: the adventure is the same one either way, and this is the target
# with no packaging in front of it.
test:
	@mkdir -p tmp
	@for part in 1 2; do \
		$(QDAAD) compile -i part$$part/game.sce -o tmp/qtest-part$$part \
			-f ddb -m pc -s vga -v 3 --part $$part || exit 1; \
		$(QRUN) -i tmp/qtest-part$$part.ddb \
			-T $(TESTS)/part$$part-extended.qtest || exit 1; \
	done

clean:
	@rm -rf tmp dist
