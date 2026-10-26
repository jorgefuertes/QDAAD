#!/usr/bin/env bash
#
# Decompiles every edition of the five adventures we can read, then holds the
# editions against one another.
#
# This is the check that matters. The binaries compared have nothing in common —
# different byte order, different header size, different offsets throughout —
# and still have to converge on the same text. Break the byte order deduction,
# the token table or the pointer arithmetic and one of the pairs stops matching.
#
# Run it from anywhere: make decomp-all, or script/decomp-all.sh.

set -uo pipefail

cd "$(dirname "${BASH_SOURCE[0]}")/.."

readonly DECOMP_DIR="work/decomp"

# Mirrors graphicsDir and listingsDir in cmd/qundaad/decompiler/assets.go. The
# first holds what shipped alongside the database rather than source, so it is
# left out of the comparisons: a different character set and a different picture
# format on every machine. The listings under it are the exception, and are
# compared on their own further down.
readonly GRAPHICS_DIR="gfx"
readonly LISTINGS_DIR="plot"

# The corpus lives in its own file, because more than one harness walks it.
source "script/editions.sh"

# Pairs that have to decompile to exactly the same source. Left out of the
# comparison: game.sce, which names the file it was read from and so differs by
# definition, and the chr and gfx directories.
#
# The tape against the disk is the strongest of these: two containers with
# nothing whatever in common — a stream of blocks written for the ear, against a
# table of sectors — arriving at the same text, byte for byte.
readonly SAME_SOURCE=(
	"ao/amiga/part1:ao/atarist/part1"
	"ao/amiga/part2:ao/atarist/part2"
	"ao/cpc/dsk/part1:ao/cpc/cdt/part1"
	"ao/cpc/dsk/part2:ao/cpc/cdt/part2"
	"ao/spectrum/dsk/part1:ao/spectrum/tzx/part1"
	"ao/spectrum/dsk/part2:ao/spectrum/tzx/part2"
	"jabato/amiga/part1:jabato/atarist/part1"
	"jabato/amiga/part2:jabato/atarist/part2"
	"jabato/pc-ega/part2:jabato/pc-cga/part2"
	"cozumel/cpc/dsk/part1:cozumel/cpc/cdt/part1"
	"cozumel/cpc/dsk/part2:cozumel/cpc/cdt/part2"
	"cozumel/spectrum/dsk/part1:cozumel/spectrum/tzx/part1"
	"cozumel/pcw/a/parte001:cozumel/pcw/a-fixed/parte001"
	"templos/cpc/dsk/part1:templos/cpc/cdt/part1"
	"templos/cpc/dsk/part2:templos/cpc/cdt/part2"
)

# Pairs whose vector drawings have to come out the same. Only the eight-bit
# Spectrum editions have any: their illustrations are strips of drawing orders
# rather than pictures, so unlike a character set or a bitmap they can be held
# to a comparison at all.
#
# The headers are left out because they name where in the container each drawing
# was read from, which is what differs between a disk and a tape by definition.
# What has to match is the orders, and all 157 of them do.
readonly SAME_DRAWINGS=(
	"ao/spectrum/dsk/part1:ao/spectrum/tzx/part1"
	"ao/spectrum/dsk/part2:ao/spectrum/tzx/part2"
)

# Pairs that have to agree on the text and the data, but not on everything.
# Machines that differ in endianness, in size and in every offset reaching one
# text is what makes this worth checking.
#
# The Amstrad PCW against the PC in Los Templos Sagrados says the most of these:
# a Z80 build for CP/M, laid at 0x100 with padding after its header, against a
# PC one of the other byte order.
readonly SAME_TEXT=(
	"ao/pc/part1:ao/amiga/part1"
	"ao/pc/part2:ao/amiga/part2"
	"ao/pc/part1:ao/atarist/part1"
	"ao/pc/part2:ao/atarist/part2"
	"ao/cpc/dsk/part1:ao/spectrum/dsk/part1"
	"jabato/pc-ega/part1:jabato/amiga/part1"
	"jabato/pc-ega/part2:jabato/amiga/part2"
	"jabato/pc-ega/part1:jabato/pc-cga/part1"
	"chichen/pc/part1:chichen/amiga/part1"
	"chichen/pc/part2:chichen/amiga/part2"
	"chichen/amiga/part1:chichen/atarist/part1"
	"chichen/amiga/part2:chichen/atarist/part2"
	"cozumel/pc/part1:cozumel/amiga/part1"
	"templos/pc/part1:templos/amiga/part1"
	"templos/pc/part2:templos/amiga/part2"
	"templos/amiga/part1:templos/atarist/part1"
	"templos/amiga/part2:templos/atarist/part2"
	"templos/pc/part1:templos/pcw/b/parte001"
	"templos/pc/part2:templos/pcw/a/parte002"
)

# The sections a SAME_TEXT pair is held to. Left out:
#   processes.sce  editions genuinely differ here: the 68000 build of La
#                  Aventura Original draws an illustration the PC one does not
#   tokens.sce     the CGA build of El Jabato was compiled without a compression
#                  table at all, and its prose still has to come out the same as
#                  the EGA one, which says more about token expansion than
#                  comparing two identical tables ever could. Where the tables
#                  do matter they are compared whole, by SAME_SOURCE.
readonly SHARED_SECTIONS=(
	vocabulary.sce sysmess.sce messages.sce
	object-text.sce location-text.sce connections.sce objects.sce
)

# Editions that describe the same world in different words.
#
# The MSX one of Chichén Itzá is the case this is for. Its prose was rewritten
# for a forty-column screen — the title screen is narrower and the room
# descriptions are cut to about half — so it cannot be held to the text of the
# PC edition. What it can be held to is the world underneath, which is the same
# game and did not change: the same objects, in the same places, with the same
# ways between rooms.
#
# That is a real check and not a weak one. Nothing about how the MSX keeps its
# database is declared anywhere; it is reconstructed from the copies its loader
# performs, and if any of that were out by a byte these would not line up.
readonly SAME_WORLD=(
	"chichen/pc/part1:chichen/msx/part1"
	"chichen/pc/part2:chichen/msx/part2"
)

readonly WORLD_SECTIONS=(objects.sce object-text.sce connections.sce)

# build compiles the decompiler once, rather than once per edition.
build() {
	local out="$1"

	go build -o "$out" cmd/qundaad/qundaad.go
}

decompile_all() {
	local qundaad="$1" edition target input

	rm -rf "$DECOMP_DIR"

	for edition in "${EDITIONS[@]}"; do
		target="${edition%%:*}"
		input="${edition#*:}"

		"$qundaad" decompile --no-binaries \
			--input "$input" --output "$DECOMP_DIR/$target" || return 1
	done

	echo "Decompiled into $DECOMP_DIR:"
	find "$DECOMP_DIR" -name game.sce | sort | sed "s|/game.sce||;s|^$DECOMP_DIR/|  |"
}

# check_same_source holds a pair to giving byte for byte the same source.
check_same_source() {
	local failed=0 pair a b

	for pair in "${SAME_SOURCE[@]}"; do
		a="${pair%%:*}"
		b="${pair#*:}"

		diff -r -q -x game.sce -x "$GRAPHICS_DIR" \
			"$DECOMP_DIR/$a" "$DECOMP_DIR/$b" > /dev/null ||
			{ echo "  FAIL $a and $b are not the same source"; failed=1; }
	done

	return "$failed"
}

# check_same_drawings holds a pair to tracing the same drawings, comparing the
# orders and not the headers that say where each was read from.
check_same_drawings() {
	local failed=0 pair a b listing name plots

	for pair in "${SAME_DRAWINGS[@]}"; do
		a="${pair%%:*}"
		b="${pair#*:}"
		plots="$GRAPHICS_DIR/$LISTINGS_DIR"

		if ! compgen -G "$DECOMP_DIR/$a/$plots/*.plot" > /dev/null; then
			echo "  FAIL $a has no drawings to compare"
			failed=1
			continue
		fi

		for listing in "$DECOMP_DIR/$a/$plots"/*.plot; do
			name="$(basename "$listing")"

			diff -q \
				<(grep -v '^;' "$listing") \
				<(grep -v '^;' "$DECOMP_DIR/$b/$plots/$name" 2> /dev/null) > /dev/null ||
				{ echo "  FAIL $a and $b disagree on $name"; failed=1; }
		done
	done

	return "$failed"
}

# check_same_world holds a pair to describing the same world, whatever words it
# uses to describe it.
check_same_world() {
	local failed=0 pair a b section

	for pair in "${SAME_WORLD[@]}"; do
		a="${pair%%:*}"
		b="${pair#*:}"

		for section in "${WORLD_SECTIONS[@]}"; do
			diff -q "$DECOMP_DIR/$a/$section" "$DECOMP_DIR/$b/$section" > /dev/null ||
				{ echo "  FAIL $a and $b disagree on $section"; failed=1; }
		done
	done

	return "$failed"
}

# check_same_text holds a pair to agreeing on the sections they share.
check_same_text() {
	local failed=0 pair a b section

	for pair in "${SAME_TEXT[@]}"; do
		a="${pair%%:*}"
		b="${pair#*:}"

		for section in "${SHARED_SECTIONS[@]}"; do
			diff -q "$DECOMP_DIR/$a/$section" "$DECOMP_DIR/$b/$section" > /dev/null ||
				{ echo "  FAIL $a and $b disagree on $section"; failed=1; }
		done
	done

	return "$failed"
}

# Where the decompiler is built. Global so that the trap that removes it can
# still see it once main has returned.
workspace=""

main() {
	local failed=0

	workspace="$(mktemp -d)"
	trap 'rm -rf "$workspace"' EXIT

	build "$workspace/qundaad" || return 1
	decompile_all "$workspace/qundaad" || return 1

	echo "Checking the editions against each other..."

	check_same_source || failed=1
	check_same_text || failed=1
	check_same_world || failed=1
	check_same_drawings || failed=1

	test "$failed" -eq 0 || return 1

	echo "  ${#SAME_SOURCE[@]} pairs give the same source whole"
	echo "  ${#SAME_TEXT[@]} pairs give the same text and data"
	echo "  ${#SAME_WORLD[@]} pairs describe the same world in other words"
	echo "  ${#SAME_DRAWINGS[@]} pairs trace the same drawings"
}

main "$@"
