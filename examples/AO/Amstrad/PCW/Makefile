# La Aventura Original for an Amstrad PCW, which is the text and no pictures.
#
# Run from this directory. The tools are run out of the repository rather than
# taken from the PATH, so a fresh checkout needs nothing installed:
#
#   make dist    one disk per part, in dist/
#   make test    both walkthroughs played, built in tmp/
#   make clean   neither directory left behind
#
# QDAAD, QPACK and QRUN are overridable, so an installed build is one
# assignment away:  make dist QDAAD=qdaad QPACK=qpack

.PHONY: dist build test clean

ROOT  := ../../../..
QDAAD ?= go run $(ROOT)/cmd/qdaad
QPACK ?= go run $(ROOT)/cmd/qpack
QRUN  ?= go run $(ROOT)/cmd/qrun

# This tree is its own, taken from +3_Extended and measured for the machine: a
# PCW is 90 columns by 32 rows where the Spectrum is 42 by 24, so every window
# number had to be moved, the way the CPC's were for its 40.
SOURCES := .

TESTS := ../../test

# One disk per part, and neither of them boots: the player starts CP/M from
# their own system disk, puts one of these in and types DAAD. It is what the
# PCW editions of 1989 asked for and what DAAD READY's PCW.BAT still does.
#
# qpack will put CP/M on a disk and start the adventure by itself — see --boot —
# and it fits on the first part. It is not used here, because then the two disks
# would not be the same kind of thing: the second part's fifty-six illustrations
# leave no room for CP/M, so it would have to be swapped in from the A> prompt
# anyway, and getting back to that prompt would mean quitting the adventure
# first. One rule for both disks is worth more than saving one word of typing.
dist: build
	@mkdir -p dist
	@for part in 1 2; do \
		$(QPACK) -f dsk -o dist/AO$$part --name AO$$part \
			tmp/build-part$$part || exit 1; \
	done

build:
	@mkdir -p tmp dist
	@for part in 1 2; do \
		$(QDAAD) compile -i $(SOURCES)/part$$part/game.sce -o tmp/build-part$$part \
			-f build -m pcw -v 3 --part $$part || exit 1; \
	done

# Built for the PC rather than the PCW because qrun plays a database and not a
# disk: the adventure is the same one either way, and this is the target with no
# packaging in front of it.
test:
	@mkdir -p tmp
	@for part in 1 2; do \
		$(QDAAD) compile -i $(SOURCES)/part$$part/game.sce -o tmp/qtest-part$$part \
			-f ddb -m pc -v 3 --part $$part || exit 1; \
		$(QRUN) -i tmp/qtest-part$$part.ddb \
			-T $(TESTS)/part$$part-extended.qtest \
			-T $(TESTS)/part$$part-death-extended.qtest || exit 1; \
	done

clean:
	@rm -rf tmp dist
