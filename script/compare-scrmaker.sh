#!/usr/bin/env bash
#
# Holds our PC pictures against SCRMAKER's, over the same PCX files.
#
# The fourth of the comparisons. compare-drf checks the two front ends against
# each other, compare-drb the two linkers, compare-daadmaker the two Spectrum
# tapes; this one checks that the .MSD files a PC release carries are the bytes
# the reference converter writes.
#
# It matters more than its size suggests. A .MSD is a header, the palette and
# the pixels, and two of those three are quietly rearranged on the way in: every
# pixel moves up sixteen places and the palette moves with it, to leave DAAD its
# own sixteen colours at the bottom for the text. Get the shift wrong and the
# picture still draws — in the wrong colours, with no error anywhere.
#
# The oracle is work/additional_sources/SCRMAKER.PAS, built here with the two
# stub units in script/scrmaker/. See that directory's README.
#
# The PCX fixtures are generated rather than shipped: DAAD READY's IMAGES/PC is
# empty, and a handful of files written to order covers more than a sample would
# — a flat screen is all runs, an alternating one is all literals, and both of
# those are the PCX encoding's edges.
#
# Needs fpc to build SCRMAKER, python3 to write the fixtures, and iconv for the
# template DAAD READY ships in ISO-8859-1.
#
# Run it from anywhere: make compare-scrmaker, or script/compare-scrmaker.sh.

set -uo pipefail

cd "$(dirname "${BASH_SOURCE[0]}")/.."

readonly SOURCE="work/additional_sources/SCRMAKER.PAS"
readonly STUBS="script/scrmaker"
readonly TEMPLATE="work/DRC/BLANK_ES.DSF"
readonly BUILD_DIR="tmp/compare-scrmaker"
readonly SCRMAKER="${BUILD_DIR}/scrmaker"
readonly QDAAD="${BUILD_DIR}/qdaad"
readonly QPACK="${BUILD_DIR}/qpack"

# The screen a VGA256 build draws on, and the height PC.BAT gives a picture:
# IMGLINES, which is 96 everywhere in the kit. The loading screen is the one
# that gets the whole screen.
readonly CANVAS_WIDTH=320
readonly CANVAS_HEIGHT=200
readonly PICTURE_LINES=96

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
require python3
require iconv

for needed in "${SOURCE}" "${TEMPLATE}"; do
	if [[ ! -f ${needed} ]]; then
		yellow "${needed} is not in the checkout: nothing to compare against, skipping."
		exit 0
	fi
done

mkdir -p "${BUILD_DIR}"

go build -o "${QDAAD}" ./cmd/qdaad || exit 1
go build -o "${QPACK}" ./cmd/qpack || exit 1

# SCRMAKER is built once and left in tmp/. fpc writes its output beside the
# source, so the source and the stubs are copied in first.
build_scrmaker() {
	if [[ -x ${SCRMAKER} ]] && [[ ${SCRMAKER} -nt ${SOURCE} ]]; then
		return 0
	fi

	echo "building scrmaker..."
	cp "${SOURCE}" "${BUILD_DIR}/scrmaker.pas"
	cp "${STUBS}"/*.pas "${BUILD_DIR}/"

	if ! (cd "${BUILD_DIR}" && fpc -O2 scrmaker.pas) >"${BUILD_DIR}/fpc.log" 2>&1; then
		red "scrmaker would not build:"
		sed -e 's/^/       /' "${BUILD_DIR}/fpc.log"
		exit 1
	fi
}

build_scrmaker

# The fixtures, and what each one is for. A PCX row is run-encoded, so what
# breaks a reader is the shape of the data and not the picture: flat is one long
# run, alternating is nothing but literals with the top bits set, and the two
# gradients sit in between.
#
# The last one uses colour 238, one below the highest a shifted palette leaves
# free, because the picture that fails is the one that uses the colour nobody
# tested.
write_fixtures() {
	local into="$1"

	python3 - "${into}" "${CANVAS_WIDTH}" "${CANVAS_HEIGHT}" <<-'PY'
		import os, struct, sys

		into, width, height = sys.argv[1], int(sys.argv[2]), int(sys.argv[3])

		def encode(row):
		    """One PCX scanline: a run is 0xC0 + count and the value, up to 63."""
		    out, at = bytearray(), 0
		    while at < len(row):
		        run = 1
		        while at + run < len(row) and row[at + run] == row[at] and run < 63:
		            run += 1
		        if run > 1 or row[at] >= 0xC0:
		            out += bytes((0xC0 | run, row[at]))
		        else:
		            out.append(row[at])
		        at += run
		    return bytes(out)

		def write(name, pixel):
		    header = bytearray(128)
		    header[0:4] = bytes((0x0A, 5, 1, 8))          # PCX, version 5, encoded, 8 bits
		    struct.pack_into('<HHHH', header, 4, 0, 0, width - 1, height - 1)
		    struct.pack_into('<HH', header, 12, 72, 72)   # dpi, which nothing reads
		    header[65] = 1                                # one plane
		    struct.pack_into('<H', header, 66, width)     # bytes a line
		    struct.pack_into('<H', header, 68, 1)         # a colour palette

		    body = bytearray()
		    for y in range(height):
		        body += encode(bytes(pixel(x, y) % 239 for x in range(width)))

		    palette = bytearray()
		    for i in range(256):
		        palette += bytes(((i * 5) % 256, (i * 11) % 256, (i * 23) % 256))

		    with open(os.path.join(into, name), 'wb') as f:
		        f.write(bytes(header) + bytes(body) + bytes((12,)) + bytes(palette))

		write('000.pcx', lambda x, y: 0)                        # every row one run
		write('001.pcx', lambda x, y: 238 if (x + y) % 2 else 0) # every pixel a literal
		write('002.pcx', lambda x, y: (x * y) // 7)              # busy
		write('003.pcx', lambda x, y: x // 4 + y // 8)           # bands
	PY
}

# The source: the kit's own empty template with our picture declaration added,
# which is the shortest thing that compiles for a PC and carries drawings.
prepare_source() {
	local work="$1"

	mkdir -p "${work}/gfx"

	write_fixtures "${work}/gfx"
	cp "${work}/gfx/003.pcx" "${work}/loading.pcx"

	# In front of the template and not after it. The declarations are comments,
	# so a source carrying them is still an ordinary source — but only the
	# header is read, everything before the first directive line, so a comment
	# at the end of the file is a comment and nothing more.
	{
		echo "; qdaad: images=gfx"
		echo "; qdaad: loading=loading.pcx"
		iconv -f ISO-8859-1 -t UTF-8 "${TEMPLATE}"
	} >"${work}/game.dsf"
}

work="${BUILD_DIR}/work"
rm -rf "${work}"
mkdir -p "${work}"

prepare_source "${work}"

if ! "${QDAAD}" compile -m pc -s vga256 -l spanish -v 3 \
	-i "${work}/game.dsf" -o "${work}/build" >"${work}/compile.log" 2>&1; then
	red "FAIL: we would not compile the template for a PC"
	sed -e 's/^/       /' "${work}/compile.log"
	exit 1
fi

# --lines is what puts our side on the reference's terms. Left out, a picture
# comes out as tall as it is, which is what this packager does by default and
# what SCRMAKER cannot do: it crops every drawing to one height for the whole
# build. See pcPicture.
if ! "${QPACK}" -f dir --name TEST --lines "${PICTURE_LINES}" \
	-o "${work}/release" "${work}/build" \
	>"${work}/pack.log" 2>&1; then
	red "FAIL: we would not pack it"
	sed -e 's/^/       /' "${work}/pack.log"
	exit 1
fi

same=0
differ=0

# compare_one converts a fixture with the reference and holds the result against
# the file in our release.
compare_one() {
	local fixture="$1" ours="$2" lines="$3"
	local name theirs

	name="$(basename "${ours}")"
	theirs="${work}/${name}.reference"

	if ! "${SCRMAKER}" pc "${fixture}" "${theirs}" \
		"${CANVAS_WIDTH}" "${lines}" 0,0 0-255 /s >"${work}/scrmaker.log" 2>&1; then
		yellow "SKIP ${name}: scrmaker would not convert it"
		sed -e 's/^/       /' "${work}/scrmaker.log"
		return 2
	fi

	if [[ ! -f ${ours} ]]; then
		red "DIFF ${name}: our release does not have it"
		return 1
	fi

	if cmp -s "${theirs}" "${ours}"; then
		green "SAME ${name} ($(wc -c <"${ours}" | tr -d ' ') bytes)"
		return 0
	fi

	local theirSize ourSize offset
	theirSize=$(wc -c <"${theirs}" | tr -d ' ')
	ourSize=$(wc -c <"${ours}" | tr -d ' ')

	red "DIFF ${name}: ${theirSize} bytes theirs, ${ourSize} ours"

	offset=$(cmp "${theirs}" "${ours}" 2>/dev/null | sed -n 's/.*char \([0-9]*\).*/\1/p')

	if [[ -n ${offset} ]]; then
		# Which of the three parts it fell in, because that is the whole
		# diagnosis: the header is geometry, the palette is the shift, and the
		# pixels are the reader.
		local at=$((offset - 1)) where
		if ((at < 16)); then
			where="the header, byte ${at}"
		elif ((at < 16 + 768)); then
			where="the palette, entry $(((at - 16) / 3))"
		else
			where="the pixels, $((at - 16 - 768)) in"
		fi

		echo "       first difference at byte ${offset}, in ${where}"
	fi

	return 1
}

tally() {
	case $1 in
	0) same=$((same + 1)) ;;
	1) differ=$((differ + 1)) ;;
	esac
}

for fixture in "${work}"/gfx/*.pcx; do
	number="$(basename "${fixture}" .pcx)"

	compare_one "${fixture}" "${work}/release/${number}.MSD" "${PICTURE_LINES}"
	tally $?
done

# And the loading screen, which is never cropped: it is the one picture with no
# text under it, so it keeps the whole screen on both sides.
compare_one "${work}/loading.pcx" "${work}/release/DAAD.MSD" "${CANVAS_HEIGHT}"
tally $?

echo
echo "${same} agree, ${differ} differ"

[[ ${differ} -eq 0 ]]
