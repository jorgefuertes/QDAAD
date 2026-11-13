#!/usr/bin/env bash
#
# Holds our ZX0 streams against Einar Saukas' own coder.
#
# This is the fourth comparison, and the one that answers a question the other
# three cannot reach. compare-drf checks the front ends against each other,
# compare-drb the back ends, compare-daadmaker the tape; all three have a
# reference tool to ask. The pictures did not: a Spectrum build compresses
# every image with ZX0, and the only check there was is that our coder's output
# is read back by our decoder. That proves the two agree. It does not prove
# either is right — a shared misreading of how an offset is coded passes the
# round trip and walks the interpreter off into memory, and the offset is
# exactly the part of the stream that cannot be read off the decoder with
# confidence.
#
# So the oracle is the coder itself. work/ZXDAAD128/DCP holds Saukas' ZX0
# source — compress.c, optimize.c, memory.c, zx0.h, (c) 2021 — beside the
# interpreter that reads what it writes. script/zx0ref/zx0ref.c is the main()
# that tree lacks, and nothing of his is copied into this repository: it is
# compiled out of work/ every time.
#
# What is compared, over every screen the examples/ trees carry:
#
#   1. His stream, our decoder, the bytes back. The one that matters.
#   2. Our stream, our decoder, the bytes back. The old round trip, kept as a
#      net under the first so that a failure says which half moved.
#   3. The sizes, as a percentage. Reported, never failed: his coder runs a
#      shortest path over the format and ours is greedy, so his streams are
#      smaller and ours are not wrong. See internal/zx0.
#
# Byte-for-byte equality is NOT expected and not asked for, which is where this
# differs from the other three. Two coders for one format are two programs, and
# only the format is the contract.
#
# The blocks are the ones a build really compresses, not raw .scr files: the
# runs of video memory a 48K tape carries, and the blanked-below screen a +3
# disk carries. That is why the comparison lives in internal/pack, behind the
# "compare" build tag, instead of in internal/zx0 where the format is.
#
# It is a tool of this repository and not a part of the compiler. Nothing in
# internal/ knows zx0ref exists; the test finds it through ZX0REF.
#
# Needs a C compiler, and work/ZXDAAD128 in the checkout.
#
# Run it from anywhere: make compare-zx0, or script/compare-zx0.sh.

set -uo pipefail

cd "$(dirname "${BASH_SOURCE[0]}")/.."

readonly DCP_DIR="work/ZXDAAD128/DCP"
readonly BUILD_DIR="tmp/compare-zx0"
readonly BRIDGE="${BUILD_DIR}/zx0ref"
readonly SOURCE="script/zx0ref/zx0ref.c"

# The parts of Saukas' coder the bridge needs. DCP.c is not among them: it is
# the reference's screen cutter, and what is wanted here is the stream.
readonly REFERENCE_PARTS=(compress.c optimize.c memory.c zx0.h)

red() { printf '\033[31m%s\033[0m\n' "$*"; }
green() { printf '\033[32m%s\033[0m\n' "$*"; }
yellow() { printf '\033[33m%s\033[0m\n' "$*"; }

require() {
	if ! command -v "$1" >/dev/null 2>&1; then
		yellow "$1 is not installed: nothing to compare against, skipping."
		exit 0
	fi
}

require cc

for part in "${REFERENCE_PARTS[@]}"; do
	if [[ ! -f ${DCP_DIR}/${part} ]]; then
		yellow "${DCP_DIR}/${part} is not in the checkout: nothing to compare against, skipping."
		exit 0
	fi
done

mkdir -p "${BUILD_DIR}"

# Built once and left in tmp/. His sources are copied beside ours rather than
# compiled in place, because work/ is someone else's checkout and this writes
# object files.
build_bridge() {
	local part

	for part in "${REFERENCE_PARTS[@]}"; do
		cp "${DCP_DIR}/${part}" "${BUILD_DIR}/"
	done

	cp "${SOURCE}" "${BUILD_DIR}/"

	echo "building the reference coder..."

	if ! (cd "${BUILD_DIR}" && cc -O2 -o zx0ref zx0ref.c compress.c optimize.c memory.c) \
		>"${BUILD_DIR}/cc.log" 2>&1; then
		red "the reference coder would not build:"
		sed -e 's/^/       /' "${BUILD_DIR}/cc.log"
		exit 1
	fi
}

build_bridge

# The comparison itself is Go: the blocks come out of the packagers, which are
# unexported, and the decoder is a package the shell cannot call. The bridge
# reaches it through ZX0REF.
if ZX0REF="$(cd "${BUILD_DIR}" && pwd)/zx0ref" \
	go test -tags compare -count 1 -timeout 600s -v ./internal/pack/ \
	>"${BUILD_DIR}/test.log" 2>&1; then
	sed -n -e 's/^ *zx0compare_test\.go:[0-9]*: /       /p' "${BUILD_DIR}/test.log"
	echo
	green "every stream agrees: the reference's streams read back, and ours do too"
	exit 0
fi

# The mode check failing is not a diff, it is the floor giving way: it says the
# streams we have been writing are not the ones the interpreter reads.
if grep -q '^--- FAIL: TestTheClassicModeIsTheOneOurDecoderReads' "${BUILD_DIR}/test.log"; then
	red "the reference's classic mode is not what our decoder reads."
	echo "       Stop here, and do not chase the rest. Every picture on every"
	echo "       tape and disk this has written is a stream our coder wrote,"
	echo "       and if the classic mode is not the format the interpreter"
	echo "       reads then none of them come out. See internal/zx0."
	exit 1
fi

red "the streams do not agree:"
grep -vE '^(=== RUN|=== PAUSE|=== CONT|    --- PASS)' "${BUILD_DIR}/test.log" | sed -e 's/^/       /'

exit 1
