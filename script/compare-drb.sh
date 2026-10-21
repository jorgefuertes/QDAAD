#!/usr/bin/env bash
#
# Holds our back end against DAAD READY's, over the same database.
#
# Where compare-drf.sh checks that the two front ends understood a source the
# same way, this one checks that the two back ends lay it out the same way, and
# there is no room for interpretation in the answer: a database is bytes, and
# either they are the same bytes or they are not.
#
# Both sides read the source with our own front end. ours.json is what qdaad
# check writes, drb builds a database out of it, qdaad builds another, and the
# two are compared. So nothing here is testing the front end — compare-drf does
# that — and a difference is the back end's, which is what makes it worth
# chasing.
#
# It is a tool of this repository and not a part of the compiler. Nothing in
# internal/ knows drb exists.
#
# Needs php, and iconv for the sources DAAD READY ships in ISO-8859-1.
#
# Run it from anywhere: make compare-drb, or script/compare-drb.sh [source...].

set -uo pipefail

cd "$(dirname "${BASH_SOURCE[0]}")/.."

readonly DRC_DIR="work/DRC"
readonly TEMPLATES_DIR="work/daad-ready/ASSETS/TEMPLATES"
readonly BUILD_DIR="tmp/compare-drb"
readonly DRB="${DRC_DIR}/src/drb.php"
readonly QDAAD="${BUILD_DIR}/qdaad"

# The date reaches the source as four symbols, so it has to be pinned or two
# runs that straddle midnight disagree about a database they built the same.
readonly WHEN="${WHEN:-2026-01-01}"

# What to build for. Three targets rather than one, because a Spectrum build
# exercises the least of this: it is little-endian and unaligned, so a ZX-only
# comparison never runs the padding or the big-endian path at all.
#
#   machine subtarget  endianness  alignment  base
#   ZX      48K        little      none       0x8400
#   PC      VGA        little      word       0x0000
#   ST      -          big         word       0x0000
readonly TARGETS=("ZX 48K" "PC VGA" "ST -")

red() { printf '\033[31m%s\033[0m\n' "$*"; }
green() { printf '\033[32m%s\033[0m\n' "$*"; }
yellow() { printf '\033[33m%s\033[0m\n' "$*"; }

require() {
	if ! command -v "$1" >/dev/null 2>&1; then
		yellow "$1 is not installed: nothing to compare against, skipping."
		exit 0
	fi
}

require php
require iconv

if [[ ! -f ${DRB} ]]; then
	yellow "${DRB} is not in the checkout: nothing to compare against, skipping."
	exit 0
fi

mkdir -p "${BUILD_DIR}"

go build -o "${QDAAD}" ./cmd/qdaad || exit 1

# sources_to_check is what was asked for, or everything we have: the template
# DRC carries, the five DAAD READY ships, and every edition we decompiled.
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
# #include paths still resolve.
#
# What the files are in now is worked out rather than assumed: DAAD READY's own
# sources are ISO-8859-1 and the ones our decompiler writes are UTF-8. A file
# that is valid UTF-8 is taken as UTF-8, which is safe in this direction — an
# ISO-8859-1 file with an accent in it is almost never valid UTF-8.
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

# where_it_falls names the section a byte at an offset belongs to, by reading
# the header of the database it came from.
#
# A difference reported as "byte 8134" says nothing. The same difference
# reported as "in the vocabulary" is most of the way to a cause, because every
# section is written by one function.
where_it_falls() {
	local file="$1" offset="$2"

	python3 - "${file}" "${offset}" <<-'PY'
		import sys

		data = open(sys.argv[1], 'rb').read()
		offset = int(sys.argv[2])

		# The header is the version, the machine, the null word, five counts,
		# and then thirteen addresses. The last of them is the end.
		names = ('compression table', 'processes', 'object texts', 'location texts',
		         'user messages', 'system messages', 'connections', 'vocabulary',
		         'object locations', 'object names', 'object weights',
		         'object attributes')

		base = int.from_bytes(data[8 + 24:8 + 26], 'little') - len(data)
		if base < 0:
		    base = int.from_bytes(data[8 + 24:8 + 26], 'big') - len(data)

		order = ('little' if data[1] >> 4 in (0, 1, 2, 3, 4, 7, 14, 15) else 'big')
		at = {n: int.from_bytes(data[8 + 2 * i:10 + 2 * i], order)
		      for i, n in enumerate(names)}

		address = base + offset
		if address < base + 34:
		    print('the header')
		    sys.exit()
		if address < base + 60:
		    print('the extern vectors')
		    sys.exit()

		# The condact blocks have no pointer of their own: they lie between the
		# connections table and the table of processes that points at them, so
		# naming them takes saying where they are rather than reading it.
		if at['connections'] <= address < at['processes']:
		    print('the condact blocks')
		    sys.exit()

		laid = sorted((v, n) for n, v in at.items() if v)
		name = 'the sections before ' + laid[0][1] if laid else 'somewhere'
		for value, section in laid:
		    if address >= value:
		        name = 'the ' + section
		print(name)
	PY
}

# compare_one builds one source for one target with both back ends.
compare_one() {
	local source="$1" machine="$2" subtarget="$3"
	local name work from json latin1 theirs ours

	name="$(echo "${source}" | sed \
		-e 's|^work/decomp/||' \
		-e 's|/game.sce$||' \
		-e 's|^work/DRC/|DRC/|' \
		-e "s|^${TEMPLATES_DIR}/|template/|")"

	work="${BUILD_DIR}/$(echo "${name}" | tr '/' '_')/${machine}"
	mkdir -p "${work}"

	from="$(dirname "${source}")"
	to_utf8 "${from}" "${work}/utf8"

	local main="${work}/utf8/$(basename "${source}")"
	json="${work}/ours.json"
	latin1="${work}/latin1.json"
	theirs="${work}/theirs.DDB"
	ours="${work}/ours.DDB"

	# Built as one list because an empty array is not something every bash
	# will expand under set -u.
	local -a ours_args=(-m "${machine}")
	local -a their_args=("${machine}")

	if [[ ${subtarget} != "-" ]]; then
		ours_args+=(-s "${subtarget}")
		their_args+=("${subtarget}")
	fi

	if ! "${QDAAD}" check --json --text=encoded "${ours_args[@]}" \
		--date "${WHEN}" -o "${json}" "${main}" >"${work}/check.log" 2>&1; then
		red "FAIL ${name} ${machine}: we would not read it"
		sed -e 's/^/       /' "${work}/check.log"
		return 1
	fi

	# drb decodes its input with utf8_encode, which means it wants the
	# half-Latin-1 that drf writes. Handing it proper UTF-8 double-encodes
	# every accent, so it is converted back on the way in. The databases are
	# identical either way once it is.
	iconv -f UTF-8 -t ISO-8859-1 "${json}" >"${latin1}" 2>/dev/null || cp "${json}" "${latin1}"

	# No compression. drb carries a table per language and prunes it, which is
	# a second thing to get right and not this one: with none, a text is its
	# own bytes and a difference can be read.
	printf '{"compression":"none","tokens":[]}' >"${work}/latin1.tok"

	if ! php "${DRB}" "${their_args[@]}" ES "${latin1}" "${theirs}" \
		>"${work}/drb.log" 2>&1; then
		yellow "SKIP ${name} ${machine}: drb would not link it (see ${work}/drb.log)"
		return 2
	fi

	if ! "${QDAAD}" compile "${ours_args[@]}" -l spanish -v 3 \
		-i "${main}" -o "${ours}" >"${work}/compile.log" 2>&1; then
		red "FAIL ${name} ${machine}: we would not link it"
		sed -e 's/^/       /' "${work}/compile.log"
		return 1
	fi

	if cmp -s "${theirs}" "${ours}"; then
		green "SAME ${name} ${machine}"
		return 0
	fi

	local theirSize ourSize offset
	theirSize=$(wc -c <"${theirs}" | tr -d ' ')
	ourSize=$(wc -c <"${ours}" | tr -d ' ')
	offset=$(cmp "${theirs}" "${ours}" 2>/dev/null | sed -n 's/.*char \([0-9]*\).*/\1/p')

	red "DIFF ${name} ${machine}: ${theirSize} bytes theirs, ${ourSize} ours"

	if [[ -n ${offset} ]]; then
		echo "       first difference at byte ${offset}, in $(where_it_falls "${theirs}" "$((offset - 1))")"
	fi

	return 1
}

same=0
differ=0
skipped=0

while IFS= read -r source; do
	[[ -z ${source} ]] && continue

	for target in "${TARGETS[@]}"; do
		read -r machine subtarget <<<"${target}"

		compare_one "${source}" "${machine}" "${subtarget}"
		case $? in
		0) same=$((same + 1)) ;;
		1) differ=$((differ + 1)) ;;
		2) skipped=$((skipped + 1)) ;;
		esac
	done
done < <(sources_to_check "$@")

echo
echo "${same} agree, ${differ} differ, ${skipped} skipped"

[[ ${differ} -eq 0 ]]
