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

Every one of them fits: 41 on part one's disk and 69 on part two's.

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

Each of the rest travels as a `.128` of its own: five bytes of geometry and the
screen compressed with ZX0, cut to its own last row with a pixel in it rather
than to a fixed height, which is what used to leave the title frame without its
bottom bar.

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
