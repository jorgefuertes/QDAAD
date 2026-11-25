# La Aventura Original for an Amstrad CPC, with the Amstrad's own pictures.
#
# Run from this directory. The tools are run out of the repository rather than
# taken from the PATH, so a fresh checkout needs nothing installed:
#
#   make dist    every disk and every tape, in dist/
#   make dsk     the 6128 disks only, every picture in the spare banks
#   make cdt     the 464 tapes only
#   make cdt128  the same tapes for a 128K machine, every picture in the banks
#   make test    both walkthroughs played, built in tmp/
#   make clean   neither directory left behind
#
# QDAAD, QPACK and QRUN are overridable, so an installed build is one
# assignment away:  make dist QDAAD=qdaad QPACK=qpack

.PHONY: dist build dsk cdt cdt128 test clean

ROOT  := ../../../..
QDAAD ?= go run $(ROOT)/cmd/qdaad
QPACK ?= go run $(ROOT)/cmd/qpack
QRUN  ?= go run $(ROOT)/cmd/qrun

# The tests are shared with the other trees and are not copies: these are the
# -extended ones, which are the same walkthroughs quoting the long texts.
TESTS := ../../test

dist: dsk cdt cdt128

# GENERIC is the picture that stands in for the ones that do not fit, and it is
# the adventure's own: the one it puts up for a location with no drawing. The
# two parts do not agree on the number — the first uses 27, the AD logotype
# among the trees, and the second 95 — so it is chosen per part below.
#
# It matters more than it sounds: PICTURE is a condition in version 3, so a
# picture that is not there fails it and abandons the rest of the entry — the
# location's own text, and the window that text is printed in. In the second
# part the entry that fails is the one at location 2, right after the access
# code, and what is left on screen is the CLS it got as far as.
GENERIC_1 ?= 27
GENERIC_2 ?= 95

# The disk is a 6128 disk and no other. Every picture goes into the four spare
# banks, compressed, so the drive is not touched again once the adventure is
# running and nothing waits for it.
#
# There used to be a 64K one beside it, which carried the pictures as files and
# fitted nine of the twenty-eight. It is gone: the nineteen that did not fit
# took the text of their locations with them, which is not a release. A machine
# without the memory loads from tape.
dsk: build
	@for part in 1 2; do \
		case $$part in 1) generic=$(GENERIC_1);; 2) generic=$(GENERIC_2);; esac; \
		$(QPACK) -f dsk --ram 128 --generic $$generic \
			--loading part$$part/gfx/loading.scr \
			-o dist/AO$$part --name AO$$part tmp/build-part$$part || exit 1; \
	done

build:
	@mkdir -p tmp dist
	@for part in 1 2; do \
		$(QDAAD) compile -i part$$part/game.sce -o tmp/build-part$$part \
			-f build -m cpc -v 3 --part $$part || exit 1; \
	done

# The tape is the loader, which carries the picture it shows, and then the
# game, and with it as many of the adventure's own pictures as fit in memory:
# a tape is read forwards and a PICTURE asks for whatever number the player
# walked into, so they cannot go on the tape itself. See README.md.
#
cdt: build
	@for part in 1 2; do \
		case $$part in 1) generic=$(GENERIC_1);; 2) generic=$(GENERIC_2);; esac; \
		$(QPACK) -f cdt --generic $$generic \
			--loading part$$part/gfx/loading.scr \
			-o dist/AO$$part --name AO$$part tmp/build-part$$part || exit 1; \
	done

# And the same tape for a machine with the other 64K — a 6128 with a cassette
# on the port, or a 464 with a memory expansion. Every picture travels, where
# the 464 tape carries the dozen that fit above the game.
cdt128: build
	@for part in 1 2; do \
		case $$part in 1) generic=$(GENERIC_1);; 2) generic=$(GENERIC_2);; esac; \
		$(QPACK) -f cdt --ram 128 --generic $$generic \
			--loading part$$part/gfx/loading.scr \
			-o dist/AO$$part-128 --name AO$$part tmp/build-part$$part || exit 1; \
	done

# Built for the PC rather than the Amstrad because qrun plays a database and
# not a disk: the adventure is the same one either way, and this is the target
# with no packaging in front of it.
test:
	@mkdir -p tmp
	@for part in 1 2; do \
		$(QDAAD) compile -i part$$part/game.sce -o tmp/qtest-part$$part \
			-f ddb -m pc -s vga -v 3 --part $$part || exit 1; \
		$(QRUN) -i tmp/qtest-part$$part.ddb \
			-T $(TESTS)/part$$part-extended.qtest \
			-T $(TESTS)/part$$part-death-extended.qtest || exit 1; \
	done

clean:
	@rm -rf tmp dist
