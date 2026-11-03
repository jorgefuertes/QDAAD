# La Aventura Original for a Spectrum +3

Two disks, one per part, loaded from the +3's own menu. Each carries the
loader, the interpreter, the database and the pictures as separate files.

A disk here is 40 tracks and **one side**, which is what the +3's built-in
drive reads. That is why there is a disk per part rather than a part per side:
a second side is one the machine cannot reach without somebody turning the disk
over.

What follows assumes `qdaad` and `qpack` are on the PATH. For where the sources
come from and what they carry corrected by hand, see
[the README above](../../README.md).

## Building

```sh
# Part 1
qdaad compile -i part1/game.sce -o build/p3-part1 -f build -m zx -s plus3 -v 3 --part 1
qpack -f dsk -o AO1 --name AO1 build/p3-part1     # AO1.dsk

# Part 2
qdaad compile -i part2/game.sce -o build/p3-part2 -f build -m zx -s plus3 -v 3 --part 2
qpack -f dsk -o AO2 --name AO2 build/p3-part2     # AO2.dsk
```

The loader has to be called `DISK`, with no extension, which is what a +3 looks
for when it boots a disk from its menu. `qpack` sees to that.

## 128 names, which is what made the pictures fit

A picture is a file of its own, opened by name when a `PICTURE` names it, so an
adventure of 69 pictures wants 73 names on its disk. Part two could not have
them, on a disk with 81 KB going spare: what ran out was names and not room.

How many a +3DOS directory holds is **whatever the disk's own specification
says**, and the specification is sixteen bytes in the first sector of track 0.
A disk in the CPC's DATA format — which is what these were, and what the CPC
disks still are — has no specification at all: it is recognised by its sector
numbering, nothing on it says how big the directory is, and the answer is 64
with no arguing with it.

So these disks are the +3's own format now, which is the one La Aventura
Original's own disk uses: sector IDs from 1 rather than 0xC1, one reserved
track, and the specification in the first sector of it. The bytes are the
original's, read out of it, with one changed:

```text
00 00 28 09 02 01 03 04 2A 52
|  |  |  |  |  |  |  |  |  '- gap when formatting
|  |  |  |  |  |  |  |  '---- gap when reading and writing
|  |  |  |  |  |  |  '------- directory blocks: 2 on the original, 4 here
|  |  |  |  |  |  '---------- block size, as log2 minus seven: 1 KiB
|  |  |  |  |  '------------- reserved tracks
|  |  |  |  '---------------- sector size, same reading: 512
|  |  |  '------------------- sectors per track
|  |  '---------------------- tracks per side
|  '------------------------- sides, and the order they are written in
'---------------------------- disk type
```

Four blocks of 32-byte entries is **128 names** for two kilobytes of directory,
and it costs the reserved track as well: 175 blocks instead of 180. Part two
uses 73 of the names, so every picture it has travels as a file.

The changed byte is the one thing here not copied from a disk that already
worked, so it was run on a +3 in ZEsarUX: the menu loads the disk, a `CAT`
lists all 73 names, and part one plays from start to finish with every picture
coming up.

That `CAT` also reports **24K free**, which is the part that proves it. Part
two's directory claims 151 blocks, so the machine has read both of the fields
that moved: it took the reserved track off the total, or it would have counted
180 blocks and said 29K, and it charged four blocks for the directory rather
than two, or it would have said 26K.

A picture travels as a `.128`: five bytes of geometry and the screen compressed
with ZX0, cut to its own last row with a pixel in it rather than to a fixed
height, which is what used to leave the title frame without its bottom bar.

The disk is deliberately **not bootable**. The +3's menu reads all 512 bytes of
that first sector and, if they sum to 3 modulo 256, takes the disk for bootable
and *runs* them; the rest of ours is formatting filler, and 0xE5 is `RST 0x20`.
So the sum is checked and kept away from 3. Nothing is lost by it: the release
is started from the menu's own Loader option, which is `LOAD "DISK"`.

### What this replaced

The interpreter has an answer of its own and we used it for a while. At start-up
it looks for a **`DAAD.GRA`**, and if it is there it reads the whole file into
page 1 and keeps it; every `PICTURE` then looks in there before going near the
disk, so a buffered picture costs no directory entry.

It works in principle and it did not work in practice: a buffered picture kept
coming out as a screen full of flashing attributes. Part of that was found — the
buffered path uncompresses with interrupts enabled, where the path that reads a
picture out of its own file knows to put a `DI` in front of the decompressor —
and patching the missing `DI` in did not cure it. What the file holds is not in
doubt; there is a test that walks the index exactly as the Z80 does,
decompresses every entry and compares it with the screen it was made from. So
whatever is left is inside the interpreter.

With 128 names nothing we ship needs it, and neither disk here has a `DAAD.GRA`
or a patched interpreter. `qpack` still writes both when an adventure has more
pictures than the disk has names, because 128 is still a number an adventure can
reach — and if that ever happens, the flashing is waiting.

### The one that is left out

Part two's drawing 4 is not a scene. It is a **piece** that drawing 107 calls,
and its listing says so: `INHERIT 107,0,6,1,76,167` — it is entered with the
pen already at (76,167) and the ink and paper 107 was using. On its own it
comes out as a grating floating on a flat yellow field.

That would not matter if it sat at an unused number, which is where the rest of
the pieces are. The drawings table is one drawing per location, so a piece has
to be parked at a location nobody visits — and the authors did exactly that
everywhere else:

| | Pieces | At a location with a description |
|---|---|---|
| Part 1 | 10 | 0 |
| Part 2 | 16 | 1 |

The one is drawing 4, and location 4 is "Te arrastras por un pasadizo bajo y
lleno de cantos rodados". So `004.scr` is not in the tree. Without it the
loop's `PICTURE @38` fails there, flag 40 stays clear, and the adventure draws
its own generic picture — which is what that mechanism is for.

**This is a deliberate difference from the original, not a repair.** Everything
above says the interpreter of 1989 drew by location number and the authors
worked around it, which means it showed the floating grating too. It has not
been checked on a real tape: part two opens by asking for the password that
finishing part one gives you.

Drawings 106 and 107 still have their own `.scr` and still show the grating:
what is packaged is a trace of the whole canvas, with the `CALL` already
followed, so the piece is drawn into them.

## The loading screen

It travels as `LOADING.SCR`, a file like any other, addressed at video memory
so that it paints itself as it comes off the disk. The loader asks for it
first, quiets the ROM, loads everything else, and then waits:

```text
10 CLEAR VAL "24575"
20 BORDER 0: PAPER 0: INK 0: CLS
30 LOAD "LOADING.SCR"CODE                  the picture, to 16384
40 POKE PEEK 23633+256*PEEK 23634,VAL "111"
50 LOAD "DAAD.BIN"CODE: LOAD "PART1.DDB"CODE: LOAD "PART1.SDG"CODE
60 PAUSE VAL "150"                         three seconds to look at it
70 RANDOMIZE USR VAL "24576"
```

The `PAUSE` is the one thing here a tape does not need. A tape's last block is
tens of kilobytes at 1500 baud, so the picture is up for most of a minute
whether anybody wanted it or not; a disk reads the same bytes in an instant,
and without the wait the picture would be gone about as fast as it arrived.

`qpack --loading <file>` packages a different one, and wins over whatever the
build directory carries.

## What is coming

This is the tree where the +3 parts company with the tape. The idea is to bring
it the long descriptions from the Amiga and CPC editions, which would not fit
on a Spectrum. Today the sources are the same as the 48K ones bar how many
pictures each carries.

## Part two asks for a password

That is the original's doing: it opens with "¿Clave de acceso?", and the word
was what finishing part one gave you.
