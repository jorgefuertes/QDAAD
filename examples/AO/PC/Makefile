# La Aventura Original for a PC, which is the one release that is a directory.
#
# Run from this directory. The tools are run out of the repository rather than
# taken from the PATH, so a fresh checkout needs nothing installed:
#
#   make dist    one directory per part, in dist/
#   make test    both walkthroughs played, built in tmp/
#   make clean   neither directory left behind
#
# QDAAD, QPACK and QRUN are overridable, so an installed build is one
# assignment away:  make dist QDAAD=qdaad QPACK=qpack

.PHONY: dist build test clean

ROOT  := ../../..
QDAAD ?= go run $(ROOT)/cmd/qdaad
QPACK ?= go run $(ROOT)/cmd/qpack
QRUN  ?= go run $(ROOT)/cmd/qrun

# This tree is its own, taken from +3_Extended and measured for the machine: a
# PC in VGA256 is 53 columns by 25 rows where the Spectrum is 42 by 24, because
# the PC's font is the only proportional one DAAD has and 320 pixels hold 53 of
# its six-pixel columns. Every window number was moved for that; nothing else
# differs.
SOURCES := .

TESTS := ../test

# A directory per part and no container around it. A PC has a filesystem, so
# DAAD's release for it is the files with the names the interpreter opens —
# PCDAAD.EXE, DAAD.DDB, DAAD.FNT, the .MSD drawings — and the player mounts the
# directory in whatever DOS they have and types the name.
#
# DOSBox is not in here. DAAD READY's PC release carries dosbox.exe and two SDL
# DLLs, which is a Windows program for running a DOS program.
dist: build
	@mkdir -p dist
	@for part in 1 2; do \
		$(QPACK) -f dir -o dist/AO$$part --name AO$$part \
			tmp/build-part$$part || exit 1; \
	done

build:
	@mkdir -p tmp dist
	@for part in 1 2; do \
		$(QDAAD) compile -i $(SOURCES)/part$$part/game.sce -o tmp/build-part$$part \
			-f build -m pc -s vga256 -v 3 --part $$part || exit 1; \
	done

# The walkthroughs, which check the adventure and not the packaging: qrun plays
# a database, and the database is the same one the release carries.
test:
	@mkdir -p tmp
	@for part in 1 2; do \
		$(QDAAD) compile -i $(SOURCES)/part$$part/game.sce -o tmp/qtest-part$$part \
			-f ddb -m pc -s vga256 -v 3 --part $$part || exit 1; \
		$(QRUN) -i tmp/qtest-part$$part.ddb \
			-T $(TESTS)/part$$part-extended.qtest \
			-T $(TESTS)/part$$part-death-extended.qtest || exit 1; \
	done

clean:
	@rm -rf tmp dist
