#!/usr/bin/env bash
#
# Decompiles every edition we can read, converts it to version 3, and compiles
# it back into a database for the machine it came off.
#
# Each edition goes to its own machine and to no other. The Spectrum tape of
# La Aventura Original becomes a Spectrum database; the Amiga disk becomes an
# Amiga one. That is the whole point: an edition was cut for the machine it
# shipped on — its texts, its drawings and how much of either would fit — so
# compiling the Amiga edition for a Spectrum asks a question nobody has.
#
# What a database is not
# ----------------------
# It holds no illustrations and no character set. Those are separate blocks
# that only exist once something is packaged: on a 48K Spectrum the database
# loads at 0x8400 and the drawings at 0xF7D7, and --charset feeds the tape
# packager rather than the compiler. So "an Amiga database with its images" is
# not a file that can be written, and the images step belongs to the packaging
# harness next door.
#
# What is machine-specific about a database, and what this therefore checks:
# where it is laid, which way round its words go, how its sections are padded,
# and that the condacts the edition actually uses all compile for that machine.
#
# The decompilation keeps the fonts and the drawings even so, beside the source
# rather than in it, because that is what the packaging step will need and this
# is where it is read off the original.
#
# The Spectrum editions and their drawings
# ----------------------------------------
# Version 3 has no vector drawings. The Spectrum editions are the only vector
# ones in the whole corpus — every other machine here illustrates with bitmaps,
# which version 3 draws perfectly well — and theirs cannot come across: not as
# vectors, because there is nothing to run them, and not as bitmaps, because 41
# screens of bitmap do not fit in a 48K Spectrum beside the game.
#
# So those compile with the illustration condacts dropped, and that is not a
# shortcut. A PICTURE that finds no image still opens the graphics window and
# leaves it there, and the text comes out with the edge of it showing.
#
# Needs nothing but the repository: both tools are ours.
set -uo pipefail

cd "$(dirname "${BASH_SOURCE[0]}")/.."

# The corpus lives in its own file, because more than one harness walks it.
source "script/editions.sh"

# Its own tree, not decomp-all's. These sources are converted to version 3 and
# the harnesses that compare us with the reference compiler read the unconverted
# ones; writing both to one place would quietly change what those compare.
readonly OUT="work/decomp-v3"
readonly BIN="tmp/compile-all/bin"

# Which machine each edition belongs to, and the variant of it where the
# machine has one. Keyed by the platform segment of the edition's own name,
# which is where that fact already lives.
#
# The PC editions are CGA and EGA because that is what shipped beside them —
# La Aventura Original's PC release carries PART1.CGA — and the Spectrum ones
# are 48K because every tape and disk here is.
machine_for() {
	case "$1" in
	amiga) echo "amiga " ;;
	atarist) echo "st " ;;
	cpc) echo "cpc " ;;
	msx) echo "msx " ;;
	pc | pc-cga) echo "pc cga" ;;
	pc-ega) echo "pc ega" ;;
	pcw) echo "pcw " ;;
	spectrum) echo "zx 48k" ;;
	*) return 1 ;;
	esac
}

green() { printf '\033[32m%s\033[0m\n' "$1"; }
red() { printf '\033[31m%s\033[0m\n' "$1"; }
yellow() { printf '\033[33m%s\033[0m\n' "$1"; }

mkdir -p "${BIN}"

echo "▶ building the tools…"
go build -o "${BIN}/qundaad" ./cmd/qundaad || exit 1
go build -o "${BIN}/qdaad" ./cmd/qdaad || exit 1
echo

ok=0
failed=0
unreadable=0
FAILURES=()

for edition in "${EDITIONS[@]}"; do
	target="${edition%%:*}"
	input="${edition#*:}"

	platform="$(echo "${target}" | cut -d/ -f2)"

	if ! read -r machine subtarget <<<"$(machine_for "${platform}")"; then
		red "NO MACHINE ${target}: nothing maps the platform ${platform}"
		failed=$((failed + 1))
		continue
	fi

	if [[ ! -f "${input}" ]]; then
		yellow "SKIP ${target}: ${input} is not in the checkout"
		continue
	fi

	src="${OUT}/${target}"
	rm -rf "${src}"
	mkdir -p "${src}"

	# Vector illustrations cannot come across to version 3, so the editions
	# that have them lose the condacts that would show them. Only the Spectrum
	# ones do; see the note at the top.
	pictures=()
	if [[ "${platform}" == "spectrum" ]]; then
		pictures=(--no-pictures)
	fi

	# --v3 does nothing to a format 2 database, which already is one, so the
	# same call serves both halves of the corpus.
	if ! "${BIN}/qundaad" decompile --v3 "${pictures[@]+"${pictures[@]}"}" \
		-i "${input}" -o "${src}" >"${src}/decompile.log" 2>&1; then
		red "UNREADABLE ${target}: see ${src}/decompile.log"
		unreadable=$((unreadable + 1))
		continue
	fi

	# An image holds more than one database and each lands in a directory of
	# its own; a single database lands where it was asked for.
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
		name="$(dirname "${source}")"
		name="${name#"${OUT}"/}"

		out="$(dirname "${source}")/$(basename "${name}").ddb"

		# The database goes beside its own source, which is where somebody
		# looking at this edition will expect to find it.
		if "${BIN}/qdaad" compile -m "${machine}" ${subtarget:+-s "${subtarget}"} \
			-v 3 -f ddb -i "${source}" -o "${out}" >"${out}.log" 2>&1; then
			ok=$((ok + 1))
			green "$(printf '%-34s %-6s %8d bytes' "${name}" "${machine}${subtarget:+/${subtarget}}" \
				"$(wc -c <"${out}")")"
		else
			failed=$((failed + 1))
			FAILURES+=("${name} -> ${machine}: $(tail -1 "${out}.log")")
			red "$(printf '%-34s %-6s FAILED' "${name}" "${machine}${subtarget:+/${subtarget}}")"
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
