#!/usr/bin/env bash
#
# Holds our tape against DAAD READY's, over the same database.
#
# This is the third of the three comparisons. compare-drf checks that the two
# front ends read a source the same way; compare-drb checks that the two back
# ends lay it out the same way; this one checks that the two packagers put the
# same bytes on a tape, at the same addresses, behind the same loader.
#
# Both sides start from a database our own compiler linked, so nothing here is
# testing the linker — compare-drb does that — and a difference is the
# packaging's, which is what makes it worth chasing.
#
# The image index is not ours either: it comes from the reference's own
# pager48k.php, run over the same database. Writing our own would mean checking
# our index against our index.
#
# It is a tool of this repository and not a part of the compiler. Nothing in
# internal/ knows daadmaker exists.
#
# Needs fpc to build daadmaker, php to run the pager, and iconv for the sources
# DAAD READY ships in ISO-8859-1.
#
# Run it from anywhere: make compare-daadmaker, or
# script/compare-daadmaker.sh [source...].

set -uo pipefail

cd "$(dirname "${BASH_SOURCE[0]}")/.."

readonly DRC_DIR="work/DRC"
readonly TEMPLATES_DIR="work/daad-ready/ASSETS/TEMPLATES"
readonly ASSETS_DIR="work/daad-ready/ASSETS"
readonly BUILD_DIR="tmp/compare-daadmaker"
readonly SOURCE="${DRC_DIR}/src/daadmaker.pas"
readonly PAGER="work/daad-ready/TOOLS/DRC/pager48k.php"
readonly DAADMAKER="${BUILD_DIR}/daadmaker"
readonly QDAAD="${BUILD_DIR}/qdaad"
readonly QPACK="${BUILD_DIR}/qpack"

readonly WHEN="${WHEN:-2026-01-01}"

# daadmaker appends four bytes we deliberately do not: an empty, headerless
# data block for the picture data, written because it is always handed a
# PAGE0.TMP and that file is empty when a game has no pictures.
#
# Its checksum is garbage. SaveBlockFromBuffer sums the block with
#
#	for i := 0 to Size - 1 do checksum := checksum xor buffer[i]
#
# and Size is a word, so a size of zero counts to 65535 instead of stopping:
# the block is summed over the whole 64K buffer, most of which is whatever the
# last file read left there. A Spectrum asked to load it would report a tape
# error. Nothing asks: the loader has three LOADs and there are three blocks
# before it.
#
# So it is dropped rather than reproduced, and these are the bytes it takes.
readonly TRAILING_EMPTY_BLOCK=4

red() { printf '\033[31m%s\033[0m\n' "$*"; }
green() { printf '\033[32m%s\033[0m\n' "$*"; }
yellow() { printf '\033[33m%s\033[0m\n' "$*"; }

require() {
	if ! command -v "$1" >/dev/null 2>&1; then
		yellow "$1 is not installed: nothing to compare against, skipping."
		exit 0
	fi
}

require fpc
require php
require iconv

if [[ ! -f ${SOURCE} ]]; then
	yellow "${SOURCE} is not in the checkout: nothing to compare against, skipping."
	exit 0
fi

mkdir -p "${BUILD_DIR}"

go build -o "${QDAAD}" ./cmd/qdaad || exit 1
go build -o "${QPACK}" ./cmd/qpack || exit 1

# daadmaker is built once and left in tmp/. Its source is Pascal and untouched;
# fpc insists on writing its output beside the source, so it is copied first.
build_daadmaker() {
	if [[ -x ${DAADMAKER} ]] && [[ ${DAADMAKER} -nt ${SOURCE} ]]; then
		return 0
	fi

	echo "building daadmaker..."
	cp "${SOURCE}" "${BUILD_DIR}/daadmaker.pas"

	if ! (cd "${BUILD_DIR}" && fpc -O2 daadmaker.pas) >"${BUILD_DIR}/fpc.log" 2>&1; then
		red "daadmaker would not build:"
		sed -e 's/^/       /' "${BUILD_DIR}/fpc.log"
		exit 1
	fi
}

build_daadmaker

sources_to_check() {
	if [[ $# -gt 0 ]]; then
		printf '%s\n' "$@"
		return
	fi

	[[ -f ${DRC_DIR}/BLANK_ES.DSF ]] && echo "${DRC_DIR}/BLANK_ES.DSF"
	find "${TEMPLATES_DIR}" -name 'BLANK_*.DSF' 2>/dev/null | sort
	find work/decomp -name game.sce 2>/dev/null | sort
}

# to_utf8 copies a source directory into UTF-8, keeping the layout so that the
# #include paths still resolve. See compare-drb.sh, which does the same for the
# same reason.
to_utf8() {
	local from="$1" to="$2"

	rm -rf "${to}"
	mkdir -p "${to}"

	local file
	while IFS= read -r file; do
		if iconv -f UTF-8 -t UTF-8 "${file}" >/dev/null 2>&1; then
			cp "${file}" "${to}/$(basename "${file}")"
		else
			iconv -f ISO-8859-1 -t UTF-8 "${file}" >"${to}/$(basename "${file}")"
		fi
	done < <(find "${from}" -maxdepth 1 -type f \( -name '*.sce' -o -name '*.DSF' -o -name '*.dsf' \))
}

# where_it_falls names the part of the tape an offset belongs to, by walking
# the blocks of the tape it came from. "byte 7262" says nothing; "in the
# interpreter block" says most of it.
where_it_falls() {
	local file="$1" offset="$2"

	python3 - "${file}" "${offset}" <<-'PY'
		import sys

		data = open(sys.argv[1], 'rb').read()
		offset = int(sys.argv[2])

		at = 0
		while at + 2 <= len(data):
		    length = data[at] | data[at + 1] << 8
		    body = data[at + 2:at + 2 + length]

		    if at <= offset < at + 2 + length:
		        where = 'a length word' if offset < at + 2 else 'a block'
		        if body and body[0] == 0x00 and len(body) >= 19:
		            kind = {0: 'program', 3: 'code'}.get(body[1], '?')
		            print(f'the header of the {kind} block '
		                  f'"{body[2:12].decode("latin-1").rstrip()}"')
		        else:
		            print(f'{where} of data, {length - 2} bytes')
		        sys.exit()

		    at += 2 + length

		print('past the end of the tape')
	PY
}

# language_for picks what to build a source in.
#
# The templates say so in their names. Everything else is one of the Spanish
# adventures. German, French and Portuguese are not languages this compiler
# speaks, so they are not built rather than built wrongly.
language_for() {
	case "$(basename "$1")" in
	BLANK_EN.DSF) echo english ;;
	BLANK_ES.DSF) echo spanish ;;
	BLANK_*.DSF) echo unsupported ;;
	*) echo spanish ;;
	esac
}

compare_one() {
	local source="$1"
	local name work from main ddb theirs ours lang

	lang="$(language_for "${source}")"
	if [[ ${lang} == unsupported ]]; then
		yellow "SKIP $(basename "${source}"): not a language this compiler speaks"
		return 2
	fi

	name="$(echo "${source}" | sed \
		-e 's|^work/decomp/||' \
		-e 's|/game.sce$||' \
		-e 's|^work/DRC/|DRC/|' \
		-e "s|^${TEMPLATES_DIR}/|template/|")"

	work="${BUILD_DIR}/$(echo "${name}" | tr '/' '_')"
	mkdir -p "${work}/theirs" "${work}/ours"

	from="$(dirname "${source}")"
	to_utf8 "${from}" "${work}/utf8"

	main="${work}/utf8/$(basename "${source}")"

	# Both sides get the same database and the same name, since the name is
	# written into every block header.
	ddb="${work}/theirs/GAME.ddb"
	theirs="${work}/theirs/GAME.TAP"
	ours="${work}/ours/GAME.tap"

	if ! "${QDAAD}" compile -m zx -s 48k -l "${lang}" -v 3 -f ddb \
		-i "${main}" -o "${ddb}" >"${work}/link.log" 2>&1; then
		# An adventure too big for a Spectrum's address space has no tape on
		# either side — daadmaker refuses it too, with 'DDB exceedes RAM size'
		# — so there is nothing to compare rather than a difference to chase.
		# Chichén Itzá is the whole of this case: it was a disk game.
		if grep -q '64 KiB' "${work}/link.log"; then
			yellow "SKIP ${name}: too big for a 48K Spectrum"
			return 2
		fi

		red "FAIL ${name}: we would not link it"
		sed -e 's/^/       /' "${work}/link.log"

		return 1
	fi

	# Two commands now that compiling and packaging are two tools: qdaad writes
	# a build directory and qpack turns it into the tape.
	if ! "${QDAAD}" compile -m zx -s 48k -l "${lang}" -v 3 \
		-i "${main}" -o "${work}/build" >"${work}/compile.log" 2>&1; then
		red "FAIL ${name}: we would not compile it"
		sed -e 's/^/       /' "${work}/compile.log"
		return 1
	fi

	if ! "${QPACK}" -f tap --name "$(basename "${ours}" .tap)" \
		-o "${ours}" "${work}/build" >"${work}/pack.log" 2>&1; then
		red "FAIL ${name}: we would not pack it"
		sed -e 's/^/       /' "${work}/pack.log"
		return 1
	fi

	# The index and the empty page file come from the reference's own pager,
	# run over the same database, in the directory daadmaker will read them
	# from. It needs the file in the current directory, which is why this runs
	# in a subshell.
	if ! (cd "${work}/theirs" && php "${OLDPWD}/${PAGER}" GAME.ddb) \
		>"${work}/pager.log" 2>&1; then
		yellow "SKIP ${name}: the pager would not run (see ${work}/pager.log)"
		return 2
	fi

	# The interpreter carries its own system messages, so there is one per
	# language and ZX48.BAT picks between them the same way.
	local interpreter="internal/runtime/assets/zx48/DS48IS3.BIN"
	[[ ${lang} == english ]] && interpreter="internal/runtime/assets/zx48/DS48IE3.BIN"

	# The argument order is ZX48.BAT's, for a build with no loading screen.
	if ! (cd "${work}/theirs" && "${OLDPWD}/${DAADMAKER}" GAME.TAP \
		"${OLDPWD}/${interpreter}" GAME.ddb \
		"${OLDPWD}/${ASSETS_DIR}/CHARSET/AD8x6.CHR" INDEX.BIN \
		"${OLDPWD}/${ASSETS_DIR}/ZX/DAAD.SDG" /48) >"${work}/daadmaker.log" 2>&1; then
		yellow "SKIP ${name}: daadmaker would not pack it (see ${work}/daadmaker.log)"
		return 2
	fi

	local theirSize ourSize trimmed
	theirSize=$(wc -c <"${theirs}" | tr -d ' ')
	ourSize=$(wc -c <"${ours}" | tr -d ' ')

	# Drop the empty block daadmaker appends, and nothing else: if it is any
	# other size than the four bytes it should be, say so instead of trimming
	# whatever is there.
	if ((theirSize - ourSize != TRAILING_EMPTY_BLOCK)); then
		red "DIFF ${name}: ${theirSize} bytes theirs, ${ourSize} ours"
		echo "       expected a difference of exactly ${TRAILING_EMPTY_BLOCK} bytes, the empty page block"

		return 1
	fi

	trimmed="${work}/theirs-trimmed.TAP"
	head -c "${ourSize}" "${theirs}" >"${trimmed}"

	if cmp -s "${trimmed}" "${ours}"; then
		green "SAME ${name}"
		return 0
	fi

	local offset
	offset=$(cmp "${trimmed}" "${ours}" 2>/dev/null | sed -n 's/.*char \([0-9]*\).*/\1/p')

	red "DIFF ${name}: ${ourSize} bytes each"

	if [[ -n ${offset} ]]; then
		echo "       first difference at byte ${offset}, in $(where_it_falls "${trimmed}" "$((offset - 1))")"
	fi

	return 1
}

same=0
differ=0
skipped=0

while IFS= read -r source; do
	[[ -z ${source} ]] && continue

	compare_one "${source}"
	case $? in
	0) same=$((same + 1)) ;;
	1) differ=$((differ + 1)) ;;
	2) skipped=$((skipped + 1)) ;;
	esac
done < <(sources_to_check "$@")

echo
echo "${same} agree, ${differ} differ, ${skipped} skipped"

[[ ${differ} -eq 0 ]]
