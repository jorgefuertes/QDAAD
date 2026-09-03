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
# Run it from anywhere: make decomp-check, or script/decomp-all.sh.

set -uo pipefail

cd "$(dirname "${BASH_SOURCE[0]}")/.."

readonly DECOMP_DIR="work/decomp"

# Mirrors fontsDir and graphicsDir in cmd/qundaad/decompiler/assets.go. They
# hold what shipped alongside the database rather than source, so they are left
# out of the comparisons: a different character set and a different picture
# format on every machine.
readonly FONTS_DIR="chr"
readonly GRAPHICS_DIR="gfx"

# Every edition we can read today, as target:input. The target is where under
# tmp/decomp the source goes; the input is a database or an image holding some.
#
# A database is written into the target directory as it stands. An image is
# opened and every database in it decompiled into a directory of its own
# underneath, which is why entries naming a whole disk name only the platform.
#
# Some of these disks and none of the tapes carry a filesystem, so their
# databases are found by searching the bytes. A disk holds both parts and they
# are numbered in the order they lie; a tape holds one, which is why the tape
# entries name the part themselves.
#
# The two Amstrad PCW disks of Los Templos Sagrados are labelled the other way
# round to what they hold: disk A carries PARTE002 and disk B carries PARTE001.
# The output is named after the database, not the disk, so the pairs further
# down look crossed and are not.
#
# Left out on purpose, all for the same reason: the loader keeps the database in
# pieces, so only its header can be found where it lies.
#   La Aventura Original  commodore64
#   El Jabato             commodore64, amstradcpc, msx, and the Spectrum tapes
readonly EDITIONS=(
	"ao/pc/part1:work/aventuras/La_Aventura_Original/PC/PART1.DDB"
	"ao/pc/part2:work/aventuras/La_Aventura_Original/PC/PART2.DDB"
	"ao/amiga:work/aventuras/La_Aventura_Original/Amiga/ORIGINAL.ADF"
	"ao/atarist:work/aventuras/La_Aventura_Original/AtariST/ORIGINAL.ST"
	"ao/cpc/dsk:work/aventuras/La_Aventura_Original/AmstradCPC/ORIGINAL.DSK"
	"ao/spectrum/dsk:work/aventuras/La_Aventura_Original/ZXSpectrum/La Aventura Original.dsk"
	"ao/cpc/cdt/part1:work/aventuras/La_Aventura_Original/AmstradCPC/La aventura original (Cara A).cdt"
	"ao/cpc/cdt/part2:work/aventuras/La_Aventura_Original/AmstradCPC/La aventura original (Cara B).cdt"
	"ao/spectrum/tzx/part1:work/aventuras/La_Aventura_Original/ZXSpectrum/La Aventura Original - Part 1.tzx"
	"ao/spectrum/tzx/part2:work/aventuras/La_Aventura_Original/ZXSpectrum/La Aventura Original - Part 2.tzx"
	"jabato/pc-ega/part1:work/aventuras/El_Jabato/PC/Ega/PART1.DDB"
	"jabato/pc-ega/part2:work/aventuras/El_Jabato/PC/Ega/PART2.DDB"
	"jabato/pc-cga/part1:work/aventuras/El_Jabato/PC/CGA/PART1.DDB"
	"jabato/pc-cga/part2:work/aventuras/El_Jabato/PC/CGA/PART2.DDB"
	"jabato/amiga:work/aventuras/El_Jabato/Amiga/JABATO.ADF"
	"jabato/atarist:work/aventuras/El_Jabato/AtariST/JABATO1.ST"
	"jabato/spectrum/dsk:work/aventuras/El_Jabato/ZXSpectrum/Jabato.dsk"
	"chichen/pc/part1:work/aventuras/Chichén_Itzá/PC/PART1.DDB"
	"chichen/pc/part2:work/aventuras/Chichén_Itzá/PC/PART2.DDB"
	"chichen/amiga:work/aventuras/Chichén_Itzá/Amiga/CHICHEN.ADF"
	"chichen/atarist:work/aventuras/Chichén_Itzá/AtariST/chicitza.st"
	"cozumel/pc/part1:work/aventuras/Cozumel/PC/PART1.DDB"
	"cozumel/pc/part2:work/aventuras/Cozumel/PC/PART2.DDB"
	"cozumel/amiga:work/aventuras/Cozumel/Amiga/COZUMEL.ADF"
	"cozumel/atarist:work/aventuras/Cozumel/AtariST/Cozumel.st"
	"cozumel/cpc/dsk:work/aventuras/Cozumel/AmstradCPC/Cozumel (S) [Original] [Dinamic].dsk"
	"cozumel/cpc/cdt/part1:work/aventuras/Cozumel/AmstradCPC/COZUMEL1.CDT"
	"cozumel/cpc/cdt/part2:work/aventuras/Cozumel/AmstradCPC/COZUMEL2.CDT"
	"cozumel/spectrum/dsk:work/aventuras/Cozumel/ZXSpectrum/Cozumel.dsk"
	"cozumel/spectrum/tzx/part1:work/aventuras/Cozumel/ZXSpectrum/cozumel1_2e.tzx"
	"cozumel/spectrum/tzx/part2:work/aventuras/Cozumel/ZXSpectrum/cozumel2_2e.tzx"
	"cozumel/pcw/a:work/aventuras/Cozumel/AmstradPCW/Cozumel_A.dsk"
	"cozumel/pcw/b:work/aventuras/Cozumel/AmstradPCW/Cozumel_B.dsk"
	"templos/pc/part1:work/aventuras/Los_templos_sagrados/PC/PART1.DDB"
	"templos/pc/part2:work/aventuras/Los_templos_sagrados/PC/PART2.DDB"
	"templos/amiga:work/aventuras/Los_templos_sagrados/Amiga/Templos Sagrados, Los (1991).adf"
	"templos/atarist:work/aventuras/Los_templos_sagrados/AtariST/TEMPLOS1.ST"
	"templos/atarist:work/aventuras/Los_templos_sagrados/AtariST/TEMPLOS2.ST"
	"templos/pcw/a:work/aventuras/Los_templos_sagrados/AmstradPCW/templosa.DSK"
	"templos/pcw/b:work/aventuras/Los_templos_sagrados/AmstradPCW/templosb.dsk"
)

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

		diff -r -q -x game.sce -x "$FONTS_DIR" -x "$GRAPHICS_DIR" \
			"$DECOMP_DIR/$a" "$DECOMP_DIR/$b" > /dev/null ||
			{ echo "  FAIL $a and $b are not the same source"; failed=1; }
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

	test "$failed" -eq 0 || return 1

	echo "  ${#SAME_SOURCE[@]} pairs give the same source whole"
	echo "  ${#SAME_TEXT[@]} pairs give the same text and data"
}

main "$@"
