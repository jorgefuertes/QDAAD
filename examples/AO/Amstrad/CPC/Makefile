# La Aventura Original for an Amstrad CPC, with the Amstrad's own pictures.
#
# Run from this directory. The tools are run out of the repository rather than
# taken from the PATH, so a fresh checkout needs nothing installed:
#
#   make dist    every disk and every tape, in dist/
#   make dsk     the 6128 disks only
#   make cdt     the 464 tapes only
#   make dsk128  the same disks for a 6128, every picture in its spare banks
#   make cdt128  the same tapes for a 128K machine, likewise
#   make test    both walkthroughs played, built in tmp/
#   make clean   neither directory left behind
#
# QDAAD, QPACK and QRUN are overridable, so an installed build is one
# assignment away:  make dist QDAAD=qdaad QPACK=qpack

.PHONY: dist build dsk dsk128 cdt cdt128 test clean

ROOT  := ../../../..
QDAAD ?= go run $(ROOT)/cmd/qdaad
QPACK ?= go run $(ROOT)/cmd/qpack
QRUN  ?= go run $(ROOT)/cmd/qrun

# The tests are shared with the other trees and are not copies: these are the
# -extended ones, which are the same walkthroughs quoting the long texts.
TESTS := ../../test

dist: dsk dsk128 cdt cdt128

# The disk is the one with the pictures on it. They are files of their own and
# the interpreter opens them by name, which is also why they are what runs out
# of room: nothing compresses a CPC picture, so one of fourteen character rows
# is sixteen kilobytes of a disk with a hundred and seventy-eight.
#
# LINES is where that is traded, and only here. Most of these drawings are 111
# rows tall, but the frame the title is printed inside is 176, so there is no
# number under 184 that carries every one of them whole — and above 96 there is
# no saving either, because 96 is the most that lets two bands of a picture
# share a 2K record. So this disk takes the one trade that pays, 15 pictures
# instead of 7, and everywhere else the pictures are compressed and go whole.
# qpack says which ones did not fit either way.
LINES ?= 96

dsk: build
	@for part in 1 2; do \
		$(QPACK) -f dsk --lines $(LINES) --loading part$$part/gfx/loading.scr \
			-o dist/AO$$part --name AO$$part tmp/build-part$$part || exit 1; \
	done

# The same disk for a 6128 and no other machine. The pictures go into the four
# spare banks, compressed, and every one of them fits: the drive is not touched
# again once the adventure is running, and nothing waits for it.
dsk128: build
	@for part in 1 2; do \
		$(QPACK) -f dsk --ram 128 --loading part$$part/gfx/loading.scr \
			-o dist/AO$$part-128 --name AO$$part tmp/build-part$$part || exit 1; \
	done

build:
	@mkdir -p tmp dist
	@for part in 1 2; do \
		$(QDAAD) compile -i part$$part/game.sce -o tmp/build-part$$part \
			-f build -m cpc -v 3 --part $$part || exit 1; \
	done

# The tape is the loader, which carries the picture it shows, and then the
# game. What it does not carry is the adventure's own pictures: a tape is read
# forwards and a PICTURE asks for whatever number the player walked into.
# qpack's --tape-pictures puts them on anyway, for whoever wants to see how far
# that gets. See README.md.
cdt: build
	@for part in 1 2; do \
		$(QPACK) -f cdt --loading part$$part/gfx/loading.scr \
			-o dist/AO$$part --name AO$$part tmp/build-part$$part || exit 1; \
	done

# And the same tape for a machine with the other 64K — a 6128 with a cassette
# on the port, or a 464 with a memory expansion. Every picture travels, where
# the 464 tape carries the dozen that fit above the game.
cdt128: build
	@for part in 1 2; do \
		$(QPACK) -f cdt --ram 128 --loading part$$part/gfx/loading.scr \
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
			-T $(TESTS)/part$$part-extended.qtest || exit 1; \
	done

clean:
	@rm -rf tmp dist
