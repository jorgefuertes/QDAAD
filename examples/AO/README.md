# La Aventura Original, ported to a version 3 interpreter

La Aventura Original is Aventuras AD's, 1989 — directed by Andrés Samudio,
programmed by Manuel González, graphics by Carlos Marqués. What is here is not
the adventure as it shipped: it is its database decompiled and then
**converted** so that a version 3 interpreter will run it, which is a different
thing and worth saying in the first line.

It serves as an example with some size to it — 49 locations and 41 drawings in
part one, 109 and 69 in part two — and as a test bench: nearly everything we
know about converting version 1 to version 3 was learnt fixing this. What each
difference is, and why, is in
[07-daad-v3.md](../../docs/DAAD_v2_v3_Research/07-daad-v3.md) §11, in Spanish.

```text
ZXSpectrum/48/    README.md, part1/, part2/     two tapes
ZXSpectrum/+3/    README.md, part1/, part2/     two disks
```

The commands for each machine are in its own README. `make example-ao` builds
all four at once.

## Where each file comes from

The `.sce` files and `gfx/` are written by the decompiler, from the original
tape:

```sh
qundaad decompile --v3 \
  --input "La Aventura Original - Part 1.tzx" \
  --output <directory>
```

They do **not** come out like this, though. What is here carries corrections
made by hand, and until the decompiler knows how to make them itself, this copy
is the one that counts. The ones there are today:

- **`game.sce`** declares the font, the pictures and the loading screen:

  ```text
  ; qdaad: charset=gfx/chr/udg.chr
  ; qdaad: images=gfx/scr
  ; qdaad: loading=gfx/loading.scr
  ```

  The font of 1989 is not the one a version 3 interpreter loads, and it shows:
  the little drawings that go beside an object's name come out as italics if it
  is not loaded.

- **`v3-loop.sce`** is the game loop, which a format 1 database does not carry
  because in 1989 it lived inside the interpreter. The one the decompiler
  writes is still the "no illustrations here" version; this one draws. It also
  carries:

  - `CLEAR 40` when clearing and `SET 40` after the `DISPLAY 0`. Flag 40 is how
    the Spectrum interpreter told the adventure whether this location had a
    drawing; neither adventure of 1989 ever writes it, they only read it.
    Without putting it back, the generic picture covers every location.
  - `WINSIZE 24 42` rather than the customary `25 127`. A Spectrum has 24 rows,
    and asking for 25 sends the first line to a row that is not there.
  - The text window at row 14, which is what most of the pictures measure.
  - `NOTAT 0` on the entries that shrink the window, because of the start-up
    screen: location 0 asks its question with `PRINTAT 11 21` over the whole
    screen, and with the window already shrunk that row does not exist.

- **`gfx/loading.scr`** is the loading screen off the original tape, its `AO$`
  block of 6912 bytes. It is the same picture in both parts.

Everything else — the other `.sce` files, `udg.chr` and the `.scr` — comes out
of the decompiler untouched. Left out are the `.png` and the `.plot`: they are
derived, for looking at rather than building with.

## The two machines

Two trees and not one with two destinations, because they are going to diverge
for real: the idea for the +3 is to bring it the long descriptions from the
Amiga and CPC editions, which would not fit on a Spectrum.

Today the only difference is **how many drawings each one carries**.

| | Part 1 | Part 2 |
|---|---|---|
| Drawings it has | 41 | 69 |
| On the 48K tape | 4 | 3 |
| On the +3 disk | 41 | 60 of 69 |

About ten fit on a tape, so each part carries the ones the game needs to make
sense and the generic picture covers the rest, which is what the adventure
itself does. On part two's disk nine are left out — not for want of room, but
because a +3DOS directory holds 64 entries and the release is 73 files.
`qpack` says which ones and never cuts one short.

## What is still not right

- **A stray full stop** turns up in the text window in some locations. It is no
  message of the database — all 63 system messages and all 140 user messages
  were marked to rule them out — and it is placed inside the text window rather
  than at a fixed row of the screen.
- **The loading screen does not travel on the +3 disk.** On the tape it does.
- **The decompiler does not reproduce this copy**: everything in the list above
  has to be put back by hand after decompiling.
