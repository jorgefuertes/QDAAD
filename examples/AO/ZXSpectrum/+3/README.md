# La Aventura Original for a Spectrum +3

Two disks, one per part, bootable from the +3's own menu. Each carries the
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

## The pictures

They all fit now: 41 on part one's disk, and 68 of part two's 69 — the one
missing is left out on purpose, see below.

They used not to. A picture is a file of its own, opened by name when a
`PICTURE` names it, and a +3DOS directory holds **64 entries** — so part two's
73 files had nine with nowhere to be named, on a disk with 81 KB going spare.
What ran out was names, not room.

The way round it is the interpreter's own and we were not using it. At start-up
it looks for a **`DAAD.GRA`**, and if it is there it reads the whole file into
page 1 and keeps it; every `PICTURE` then looks in there before going near the
disk. So the pictures that fit in that page — 16 on part one, 25 on part two —
cost one directory entry between them, and draw with no disk access at all. The
smallest go in, because the point is to take as many names as possible out of
the directory.

A disk that uses the buffer also carries a **patched interpreter**. Paging a
RAM bank in ends with an EI, so whatever runs next runs interruptible with an
unusual bank at 0xC000; the path that reads a picture out of its own file knows
this and puts a DI in front of the decompressor, and the path that reads one
out of the buffer does not. It never mattered while nothing used `DAAD.GRA`.
It shows up as a picture that comes out black, or a machine that resets, at no
particular place and not every time — it depends where the 50 Hz interrupt
lands inside the decompression.

The patch is five bytes appended after the interpreter, in the kilobyte
between where it ends and where the database starts: the one call is sent
through a `DI` / `CALL` / `RET` stub. Nothing inside the image moves, so every
address in it still means what it meant.

Each of the rest travels as a `.128` of its own: five bytes of geometry and the
screen compressed with ZX0, cut to its own last row with a pixel in it rather
than to a fixed height, which is what used to leave the title frame without its
bottom bar.

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

It is in `partN/gfx/loading.scr` and **does not travel on the disk yet**: the
BASIC loader carries it on the 48K tape, and here it is still to do.

## What is coming

This is the tree where the +3 parts company with the tape. The idea is to bring
it the long descriptions from the Amiga and CPC editions, which would not fit
on a Spectrum. Today the sources are the same as the 48K ones bar how many
pictures each carries.

## Part two asks for a password

That is the original's doing: it opens with "¿Clave de acceso?", and the word
was what finishing part one gave you.
