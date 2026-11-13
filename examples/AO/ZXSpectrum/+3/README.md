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

Or the `Makefile` here, which does it from this directory and needs nothing
installed — it runs the tools out of the repository:

```sh
make dist     # both disks, in dist/
make test     # both walkthroughs played, built in tmp/
make clean    # neither directory left behind
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
and it costs the reserved track as well: 175 blocks instead of 180.

The changed byte is the one thing here not copied from a disk that already
worked, so it was run on a +3 in ZEsarUX, on the build that still carried every
drawing: the menu loads the disk, a `CAT` lists all 73 names, part one plays
from start to finish with every picture coming up, and the `CAT` reports **24K
free**.

That last figure is the part that proves it, because it is arithmetic and not
an impression. Part two's directory claimed 151 blocks, so 24 free is the
machine having read both of the fields that moved: it took the reserved track
off the total, or it would have counted 180 blocks and said 29K, and it charged
four blocks for the directory rather than two, or it would have said 26K.

**And then the disks stopped needing it.** Dropping the drawings nothing can
ask for — see below — took part two from 73 names to 46 and part one to 33, and
46 would have fitted in a DATA disk's 64. So the 128 is headroom now rather
than a necessity. It stays: it is the +3's own format, it is measured, and an
adventure that wants a picture for each of a hundred locations is not a strange
adventure.

| | Names | Blocks used | Free |
|---|---|---|---|
| Part 1 | 33 of 128 | 88 of 175 | 87K |
| Part 2 | 46 of 128 | 116 of 175 | 59K |

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

Nothing we ship needs it twice over now — 128 names, and 46 wanted — and
neither disk here has a `DAAD.GRA` or a patched interpreter. `qpack` still
writes both when an adventure has more pictures than the disk has names,
because 128 is still a number an adventure can reach, and if that ever happens
the flashing is waiting.

## What is left out, and why

Part one packages **28** of its 41 drawings and part two **41** of its 69. The
rest are not missing: they are drawings nothing can ask for.

### The library

Most of the drawings in these databases are not scenes. They are pieces and
backgrounds that other drawings `CALL` — a grating, a rock face, a figure — and
on the machine of 1989 that mattered, because the engine followed the `CALL` at
draw time and a piece saved the bytes of drawing it twice.

What we package is each drawing **already traced with its calls followed**, so
every piece is inside the drawings that use it and nothing ever opens one by
name. On a disk that made them pure cost: 13 names and kilobytes in part one,
27 in part two, better than a third of what was going on.

A drawing is kept when the adventure can ask for it, which is one of two ways:

- an explicit `PICTURE n` in the processes — part one names 27, 44, 46, 47 and
  48; part two names 1, 43, 82, 95, 106 and 107;
- `PICTURE @38`, the loop drawing the picture of wherever the player is, so any
  location the player can reach.

Reachable is taken widely on purpose — described, or with an exit, or the far
end of somebody's exit, or the target of a `GOTO` — because keeping one too
many costs a kilobyte and dropping one too few costs a location its picture.
The one way round it the adventure has is part one's `COPYFF 55 38`, which puts
the player somewhere to drop an object down it; the `AUTOD` drops at flag 38
and the `GOTO 18` two condacts later puts them back, inside the one entry, so
no turn ever ends there.

`TestThePlus3TreesCarryWhatTheAdventureCanAskFor` works all of that out from
the sources and fails if a tree carries a drawing nothing asks for, or is
missing one a `PICTURE n` names.

Three of them were not even library: part one's 33 and part two's 31 and 101
are called by nothing and sit at no location. They are an unfinished sketch, a
plain blue field, and half a boat.

### And one that is a deliberate difference

Part two's drawing 4 is a **piece**, and its listing says so:
`INHERIT 107,0,6,1,76,167` — it is entered with the pen already at (76,167) and
the ink and paper 107 was using. On its own it comes out as a grating floating
on a flat yellow field.

That would not matter if it sat at an unused number, which is where the rest of
the pieces are: a piece has to be parked at a location nobody visits, and the
authors did exactly that everywhere else. Location 4 is "Te arrastras por un
pasadizo bajo y lleno de cantos rodados", so this one they missed. Without
`004.scr` the loop's `PICTURE @38` fails there, flag 40 stays clear, and the
adventure draws its own generic picture — which is what that mechanism is for.

**This is a deliberate difference from the original, not a repair.** The
interpreter of 1989 drew by location number and the authors worked around it,
which means it showed the floating grating too. Drawings 106 and 107 still have
their own `.scr` and still show the grating, because what is packaged is a
trace of the whole canvas with the `CALL` already followed.

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

## The long texts are next door

The sources here are the same as the 48K ones bar how many pictures each
carries, and they stay that way: this is the tape's adventure on a disk.

The +3 release that parts company with the tape is
[`../+3_Extended`](../+3_Extended/README.md), which is these disks with the
long descriptions of the Amiga, PC and Atari ST editions in them — the ones
that would not fit on a Spectrum of 1989.

## Part two asks for a password

That is the original's doing: it opens with "¿Clave de acceso?", and the word
was what finishing part one gave you.
