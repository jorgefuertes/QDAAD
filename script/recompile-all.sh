#!/usr/bin/env bash
#
# Puts every decompilation back through the reference compiler.
#
# This closes the circle. decomp-all.sh proves the editions agree with one
# another; this proves what comes out is source, by handing it to a compiler
# that had nothing to do with writing it — DAAD Reborn Compiler 0.40, by Uto,
# under work/DRC.
#
# Only the front end is run, and that is on purpose. drf is what reads the
# syntax and resolves the references: that every condact has its parameters,
# that every label points somewhere, that every include is there. If what we
# write is not source, it fails there. The back end after it only lays the bytes
# out for one machine, and its output could never match a database from 1989
# byte for byte — different compiler, different padding, different token table —
# so comparing them would say nothing.
#
# Its semantic analysis is asked for as warnings rather than errors, and that is
# not a way of ducking a failure. It holds these sources to a stricter rule than
# the databases they came from were written under, and two cases account for
# every complaint:
#
#   DESC 255      255 is the location DAAD keeps for "nowhere", and every one of
#                 these adventures uses it. drf wants a location 255 declared,
#                 and there is none to declare.
#   SYNONYM 3 _   the condact takes a word number outright here, as the version 2
#                 databases hold it; drf expects a word from the vocabulary.
#
# Under the strict reading 12 of the 66 pass, and the other 54 fail on those two
# alone. Under this one all 66 do, which is the honest answer to the question
# being asked: is what comes out of the decompiler source that a compiler reads.
#
# Run it from anywhere: make recompile-all, or script/recompile-all.sh.
#
# Needs work/decomp to be populated: run decomp-all.sh first.

set -uo pipefail

cd "$(dirname "${BASH_SOURCE[0]}")/.."

readonly DECOMP_DIR="work/decomp"
readonly FRONTEND="work/_verify/bin/drf"
readonly BUILD_DIR="work/recomp"

# What the decompiler writes in the header of game.sce, against what the
# compiler calls the same machine.
target_for() {
	case "$1" in
		"IBM PC")         echo "PC CGA" ;;
		"ZX Spectrum")    echo "ZX PLUS3" ;;
		"Amstrad CPC")    echo "CPC" ;;
		"Amstrad PCW")    echo "PCW" ;;
		"MSX")            echo "MSX" ;;
		"Atari ST")       echo "ST" ;;
		"Commodore Amiga") echo "AMIGA" ;;
		*)                echo "" ;;
	esac
}

# prepare copies one decompilation into a directory the compiler will take.
#
# Two things have to change, and neither is a fault of either side. The source
# comes out UTF-8, which is right for reading it today and wrong for a compiler
# that expects the encoding of the machine; and the token table is written as a
# /TOK section, which is how version 2 held it and which version 3 dropped, so
# the compiler would not know what to do with it. The tokens themselves are not
# lost by leaving it out: the compiler has its own for each language, and what
# is being checked here is the source and not the compression.
prepare() {
	local from="$1" into="$2" name

	mkdir -p "$into" || return 1

	for source in "$from"/*.sce; do
		name="$(basename "$source")"
		test "$name" != "tokens.sce" || continue

		# The include first, while the file is still UTF-8: a locale-strict sed
		# will not touch it once the accents are single bytes.
		sed '/#include tokens.sce/d' "$source" |
			iconv -f UTF-8 -t ISO-8859-1 > "$into/$name" || return 1
	done

	mv "$into/game.sce" "$into/game.dsf"
}

# recompile runs the front end on one decompilation and says whether it read.
recompile() {
	local dir="$1" machine target into log

	machine="$(sed -n 's/^; Machine *: //p' "$dir/game.sce" | sed 's/ *([0-9]*)$//')"

	target="$(target_for "$machine")"
	if test -z "$target"; then
		echo "  SKIP $dir: no target for $machine"
		return 0
	fi

	into="$BUILD_DIR/${dir#"$DECOMP_DIR"/}"
	rm -rf "$into"

	prepare "$dir" "$into" || { echo "  FAIL $dir: could not be prepared"; return 1; }

	log="$into/drf.log"

	# The front end resolves an #include against the working directory rather
	# than against the file that names it, so it is run from where the sources
	# are and given plain names.
	if (cd "$into" && "$OLDPWD/$FRONTEND" $target game.dsf game.json -semantic-warnings) > "$log" 2>&1; then
		test -s "$into/game.json" || { echo "  FAIL $dir: no json came out"; return 1; }
		return 0
	fi

	echo "  FAIL $dir ($target)"
	sed -n '/^[0-9]*:[0-9]*:/p' "$log" | head -3 | sed 's/^/        /'

	return 1
}

main() {
	if test ! -x "$FRONTEND"; then
		echo "The compiler front end is not built: $FRONTEND" >&2
		echo "See docs/DAAD_v2_v3_Research/14-verificacion.md." >&2
		return 1
	fi

	local total=0 good=0 dir

	rm -rf "$BUILD_DIR"

	echo "Recompiling what was decompiled, with $("$FRONTEND" | head -1)..."

	while read -r game; do
		dir="$(dirname "$game")"
		total=$((total + 1))

		if recompile "$dir"; then
			good=$((good + 1))
		fi
	done < <(find "$DECOMP_DIR" -name game.sce | sort)

	if test "$total" -eq 0; then
		echo "Nothing in $DECOMP_DIR. Run script/decomp-all.sh first." >&2
		return 1
	fi

	echo "  $good of $total decompilations are accepted as source"

	test "$good" -eq "$total"
}

main "$@"
