#!/usr/bin/env bash
#
# Compiles every edition we can read into a version 3 database, for every
# machine we target.
#
# This asks a different question from the harnesses next to it. compare-drf and
# compare-drb hold our compiler against the reference one; recompile-all hands
# our sources to a compiler that had nothing to do with writing them. All three
# are about agreement. This one is about reach: take an adventure off its 1989
# tape, convert it to version 3, and see whether it compiles for each of the
# fourteen machines — a Commodore 64 and a PCW included, which never had a
# version 3 interpreter and are here because the compiler offers them.
#
# It stops at the database and does not package anything. A tape or a disk is a
# machine's own business and has its own harness; what is being asked here is
# whether the source is whole and the back end can lay it out, which is what a
# .ddb answers.
#
# Needs nothing but the repository: both tools are ours.
set -uo pipefail

cd "$(dirname "${BASH_SOURCE[0]}")/.."

# The corpus lives in its own file, because more than one harness walks it.
source "script/editions.sh"

readonly OUT="tmp/compile-all"
readonly BIN="${OUT}/bin"

# Every machine the compiler takes. Asked for rather than written down, so that
# a target added to the compiler is a target asked about here.
#
# Read with a loop and not with mapfile: macOS ships bash 3.2, which has not
# got it, and a harness that only runs on a machine with a newer bash installed
# is a harness nobody runs.
MACHINES=()
while read -r machine; do
	[[ -n "${machine}" ]] && MACHINES+=("${machine}")
done < <(go run ./cmd/qdaad compile --help 2>/dev/null |
	sed -n 's/.*Valid values are: \(.*\)$/\1/p' | tr ',' '\n' |
	sed 's/ or / /; s/^ *//; s/ *$//' | tr ' ' '\n')

if [[ ${#MACHINES[@]} -eq 0 ]]; then
	echo "could not ask the compiler which machines it takes" >&2
	exit 1
fi

green() { printf '\033[32m%s\033[0m\n' "$1"; }
red() { printf '\033[31m%s\033[0m\n' "$1"; }
yellow() { printf '\033[33m%s\033[0m\n' "$1"; }

mkdir -p "${BIN}"

# Built once. Fifty editions times fourteen machines is seven hundred
# compilations, and paying for a link of the compiler on each would be most of
# the wall clock.
echo "▶ building the tools…"
go build -o "${BIN}/qundaad" ./cmd/qundaad || exit 1
go build -o "${BIN}/qdaad" ./cmd/qdaad || exit 1

echo "▶ ${#MACHINES[@]} machines: ${MACHINES[*]}"
echo

ok=0
failed=0
unreadable=0
declare -a FAILURES=()

for edition in "${EDITIONS[@]}"; do
	target="${edition%%:*}"
	input="${edition#*:}"

	if [[ ! -f "${input}" ]]; then
		yellow "SKIP ${target}: ${input} is not in the checkout"
		continue
	fi

	src="${OUT}/src/${target}"
	rm -rf "${src}"
	mkdir -p "${src}"

	# Converted on the way out, which is where the conversion belongs: --v3
	# renumbers the processes, writes the loop a format 1 database has not got,
	# and turns the condacts that changed meaning. -B leaves the fonts and the
	# drawings where they are; none of them affect whether this compiles.
	if ! "${BIN}/qundaad" decompile --v3 -B -i "${input}" -o "${src}" \
		>"${src}/decompile.log" 2>&1; then
		red "UNREADABLE ${target}: see ${src}/decompile.log"
		unreadable=$((unreadable + 1))
		continue
	fi

	# An image holds more than one database, so each lands in a directory of
	# its own; a single database lands where it was asked for. Either way what
	# is compiled is every game.sce underneath.
	sources=()
	while read -r found; do
		[[ -n "${found}" ]] && sources+=("${found}")
	done < <(find "${src}" -name 'game.sce' | sort)

	if [[ ${#sources[@]} -eq 0 ]]; then
		red "EMPTY ${target}: nothing came out"
		unreadable=$((unreadable + 1))
		continue
	fi

	for source in "${sources[@]}"; do
		part="${source#"${src}"/}"
		part="${part%/game.sce}"
		[[ "${part}" == "game.sce" ]] && part=""

		name="${target}${part:+/${part}}"
		line=""

		for machine in "${MACHINES[@]}"; do
			out="${OUT}/ddb/${name}/${machine}.ddb"
			mkdir -p "$(dirname "${out}")"

			if "${BIN}/qdaad" compile -m "${machine}" -v 3 -f ddb \
				-i "${source}" -o "${out}" >"${out}.log" 2>&1; then
				ok=$((ok + 1))
				line+="."
			else
				failed=$((failed + 1))
				line+="✗"
				FAILURES+=("${name} ${machine}: $(tail -1 "${out}.log")")
			fi
		done

		if [[ "${line}" == *✗* ]]; then
			red "${line}  ${name}"
		else
			green "${line}  ${name}"
		fi
	done
done

echo
if [[ ${#FAILURES[@]} -gt 0 ]]; then
	echo "What failed:"
	printf '  %s\n' "${FAILURES[@]}"
	echo
fi

echo "${ok} compiled, ${failed} failed, ${unreadable} editions unreadable"

[[ ${failed} -eq 0 && ${unreadable} -eq 0 ]]
