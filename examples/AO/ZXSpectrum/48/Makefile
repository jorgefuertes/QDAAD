# La Aventura Original for a 48K Spectrum.
#
# Run from this directory. The tools are run out of the repository rather than
# taken from the PATH, so a fresh checkout needs nothing installed:
#
#   make dist    both tapes, as TAP and as TZX, in dist/
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

# Shared with the +3 tree, which carries the same sources.
TESTS := ../../test

# Both tape formats, because they are the same blocks and the choice is the
# loader's: the TZX is the TAP with a signature and a pause after each block.
dist:
	@mkdir -p tmp dist
	@for part in 1 2; do \
		$(QDAAD) compile -i part$$part/game.sce -o tmp/build-part$$part \
			-f build -m zx -s 48k -v 3 --part $$part || exit 1; \
		$(QPACK) -f tap -o dist/AO$$part --name AO$$part tmp/build-part$$part || exit 1; \
		$(QPACK) -f tzx -o dist/AO$$part --name AO$$part tmp/build-part$$part || exit 1; \
	done

# Built for the PC rather than the Spectrum because qrun plays a database and
# not a tape: the adventure is the same one either way, and this is the target
# with no packaging in front of it.
test:
	@mkdir -p tmp
	@for part in 1 2; do \
		$(QDAAD) compile -i part$$part/game.sce -o tmp/qtest-part$$part \
			-f ddb -m pc -s vga -v 3 --part $$part || exit 1; \
		$(QRUN) -i tmp/qtest-part$$part.ddb -T $(TESTS)/part$$part.qtest || exit 1; \
	done

clean:
	@rm -rf tmp dist
