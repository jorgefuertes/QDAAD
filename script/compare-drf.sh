#!/usr/bin/env bash
#
# Holds our front end against DAAD READY's, over the same source.
#
# Reading a source without an error only proves it was read. This reads it
# twice, with two implementations that share no code, and compares what each
# one understood: every word, every message, every condact and every parameter.
# A wrong opcode or a parameter resolved down the wrong path shows up here and
# nowhere else.
#
# It is a tool of this repository and not a part of the compiler. Nothing in
# internal/ knows drf exists, and qdaad never runs it: the comparison lives
# here so that building a game needs nothing but our own binary.
#
# Needs fpc to build drf, and iconv because DAAD READY's sources are
# ISO-8859-1 and ours only reads UTF-8. Without either it says so and stops.
#
# Run it from anywhere: make compare-drf, or script/compare-drf.sh [source...].
# With no arguments it takes the template and every decompiled edition.

set -uo pipefail

cd "$(dirname "${BASH_SOURCE[0]}")/.."

readonly DRC_DIR="work/DRC"
readonly BUILD_DIR="tmp/compare-drf"
readonly DRF="${BUILD_DIR}/drf"
readonly QDAAD="${BUILD_DIR}/qdaad"
readonly COMPARE="${BUILD_DIR}/jsoncmp"

# What the two are told to compile for. It has to be the same on both sides:
# the target seeds symbols, a source branches on them, and comparing a Spectrum
# build against an Amiga one would report differences that are not faults.
readonly TARGET="${TARGET:-ZX}"
readonly SUBTARGET="${SUBTARGET:-128K}"

# The date reaches the source as four symbols, so it has to be pinned or the
# two runs disagree whenever they straddle midnight.
readonly WHEN="${WHEN:-2026-01-01}"

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
require iconv

mkdir -p "${BUILD_DIR}"

# drf is built once and left in tmp/. Its own sources are Pascal and untouched;
# we only ever read them.
build_drf() {
	if [[ -x ${DRF} ]] && [[ ${DRF} -nt ${DRC_DIR}/src/drf.pas ]]; then
		return 0
	fi

	echo "building drf..."
	if ! (cd "${DRC_DIR}/src" && fpc -Mobjfpc -O2 -o"../../../${DRF}" drf.pas) >"${BUILD_DIR}/fpc.log" 2>&1; then
		red "drf did not build; see ${BUILD_DIR}/fpc.log"
		exit 1
	fi
}

build_ours() {
	go build -o "${QDAAD}" ./cmd/qdaad || exit 1
	go build -o "${COMPARE}" ./cmd/jsoncmp || exit 1
}

# sources_to_check is what was asked for, or everything we have.
sources_to_check() {
	if [[ $# -gt 0 ]]; then
		printf '%s\n' "$@"
		return
	fi

	[[ -f ${DRC_DIR}/BLANK_ES.DSF ]] && echo "${DRC_DIR}/BLANK_ES.DSF"
	find work/decomp -name game.sce 2>/dev/null | sort
}

# compare_one runs both front ends over one source and reports.
#
# Each side gets the tree in the encoding it reads: drf only understands
# ISO-8859-1 and ours only understands UTF-8. Both copies say the same thing,
# and it is the one difference between the two runs.
compare_one() {
	local source="$1"
	local name work theirs ours from

	name="$(echo "${source}" | sed -e 's|^work/decomp/||' -e 's|/game.sce$||' -e 's|^work/DRC/||')"
	work="${BUILD_DIR}/$(echo "${name}" | tr '/' '_')"
	mkdir -p "${work}"

	theirs="${work}/theirs.json"
	ours="${work}/ours.json"
	from="$(dirname "${source}")"

	convert_tree "${from}" "${work}/latin1" "ISO-8859-1"
	convert_tree "${from}" "${work}/utf8" "UTF-8"
	drop_tok "${work}/latin1"

	# drf wants to be run beside the source: #include paths are relative.
	# Its arguments go target, subtarget, input, output — see drf.pas:325-337.
	# Only four machines take a subtarget there, and passing one to any other
	# would be read as the input file name.
	local -a args=("${TARGET}")
	case "${TARGET}" in
	ZX | ZX81 | MSX2 | PC) args+=("${SUBTARGET}") ;;
	esac

	local here
	here="$(pwd)"

	# -no-semantic is not a way of dodging failures, it is what makes the
	# comparison the right one. drf refuses a "MESSAGE 40" when there are
	# thirty-nine messages, and the released adventures are full of such
	# things — they ran anyway, and the decompiled sources say so faithfully.
	# What is being compared here is what each front end understood the source
	# to say, not whether the source is a well-behaved one.
	if ! (cd "${work}/latin1" &&
		"${here}/${DRF}" "${args[@]}" "$(basename "${source}")" \
			"${here}/${theirs}" -v3 -no-semantic) >"${work}/drf.log" 2>&1; then
		yellow "SKIP ${name}: drf would not read it (see ${work}/drf.log)"
		return 2
	fi

	if ! "${QDAAD}" check --json -m "${TARGET}" -s "${SUBTARGET}" \
		--date "${WHEN}" -o "${ours}" \
		"${work}/utf8/$(basename "${source}")" >"${work}/qdaad.log" 2>&1; then
		red "FAIL ${name}: we would not read it"
		sed -e 's/^/       /' "${work}/qdaad.log"
		return 1
	fi

	if ! "${COMPARE}" "${theirs}" "${ours}" >"${work}/diff.txt" 2>&1; then
		red "DIFF ${name}"
		sed -e 's/^/       /' "${work}/diff.txt"
		return 1
	fi

	green "SAME ${name}"
	return 0
}

# convert_tree copies a source directory into one encoding, keeping the layout
# so that the #include paths still resolve.
#
# What the files are in now is worked out rather than assumed: DAAD READY's own
# sources are ISO-8859-1 and the ones our decompiler writes are UTF-8, and both
# are compared here. A file that is valid UTF-8 is taken as UTF-8, which is
# safe in this direction — an ISO-8859-1 file with an accent in it is almost
# never valid UTF-8, and one without an accent converts to itself either way.
convert_tree() {
	local from="$1" to="$2" want="$3"

	rm -rf "${to}"
	mkdir -p "${to}"

	local file have
	while IFS= read -r file; do
		if iconv -f UTF-8 -t UTF-8 "${file}" >/dev/null 2>&1; then
			have="UTF-8"
		else
			have="ISO-8859-1"
		fi

		if [[ ${have} == "${want}" ]]; then
			cp "${file}" "${to}/$(basename "${file}")"
		else
			iconv -f "${have}" -t "${want}" "${file}" >"${to}/$(basename "${file}")"
		fi
	done < <(find "${from}" -maxdepth 1 -type f \( -name '*.sce' -o -name '*.DSF' -o -name '*.dsf' \))
}

# drop_tok takes the token table out of the copy drf reads.
#
# /TOK is the compression dictionary of the 1991 format. Our decompiler writes
# it because it is worth having on record, and both front ends ignore it: ours
# skips the section, and drf does not know it at all — it stops at the first
# line, which is where the escaped characters of a compressed token live.
#
# Taking it out is therefore not hiding anything. There is nothing in it that
# either side would put in a database, and leaving it in only means drf refuses
# to read sixty of the sixty-six sources.
drop_tok() {
	local dir="$1"

	local file
	while IFS= read -r file; do
		# The first thing that is not a comment or a blank line tells us what
		# the file is.
		if [[ $(grep -m1 -v -e '^[[:space:]]*;' -e '^[[:space:]]*$' "${file}") != "/TOK"* ]]; then
			continue
		fi

		local name
		name="$(basename "${file}")"
		rm -f "${file}"

		# And the #include that named it has to go with it.
		grep -rl "#include[[:space:]]*${name}" "${dir}" 2>/dev/null |
			while IFS= read -r includer; do
				sed -i.bak -e "s|^[[:space:]]*#include[[:space:]]*${name}.*$|; ${name} removed: /TOK is not read|" \
					"${includer}"
				rm -f "${includer}.bak"
			done
	done < <(find "${dir}" -maxdepth 1 -type f -name '*.sce')
}

build_drf
build_ours

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
