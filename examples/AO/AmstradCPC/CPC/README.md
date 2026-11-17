# La Aventura Original — Amstrad CPC

The +3 Extended tree with the Amstrad's own drawings: the same text, the
Amstrad's pictures, the Amstrad's font.

```sh
make dist    # every disk and every tape, in dist/
make dsk     # the 6128 disks only
make cdt     # the 464 tapes only
make dsk128  # the same disks with every picture in a 6128's spare banks
make test    # both walkthroughs played
```

or, without make:

```sh
qdaad compile -i part1/game.sce -o tmp/build-part1 -f build -m cpc -v 3 --part 1
qpack -f dsk --lines 96 --loading part1/gfx/loading.scr \
      -o dist/AO1 --name AO1 tmp/build-part1
qpack -f cdt --loading part1/gfx/loading.scr \
      -o dist/AO1 --name AO1 tmp/build-part1
```

The disk is a 180K DATA-format `.dsk` and the game starts with `RUN"AO1"`. The
tape is a `.cdt` and starts the same way.

## The picture it shows while it loads

`--loading` is what asks for one, and it is decompiled out of the original like
everything else here: `gfx/loading.scr` is the 1989 edition's own presentation
screen — mode 0, the adventurer and the creature in the cave, signed
J.A.DARDER — with `gfx/loading.pal` beside it holding the sixteen pens and the
border. The same bytes come off the tape and off the disk, which is what says
they were read right.

Asking for one turns the release from one file into three: the loader, the
picture and the game. The loader is what `RUN"AO1"` starts and it is a single
page, up at the top of memory where nothing else wants to be; it sets mode 0
and the palette, drops the picture straight onto the display, and then pulls
the rest in underneath it, so the picture is up for as long as the loading
takes.

It also tells the cassette manager to be quiet before any of that. Left noisy,
the firmware prints "Loading AO1.GAM block 3" over the picture it has just put
up, and stops between files to ask for a key; quiet, it prints nothing and does
not stop.

## Where the pictures come from

They are not the Spectrum's converted. The Amstrad edition of 1989 carries its
own drawings — the same art, redrawn for a machine with four colours in a
palette instead of two to a character cell — and they are decompiled out of it:

```sh
qundaad decompile -i ORIGINAL.DSK -o cpc
```

writes `gfx/scr/nnn.scr`, the mode 1 screen as video memory holds it, and
`gfx/scr/nnn.pal`, the four inks it is read through. What is in this tree is
those two files for each of the drawings the +3 tree carries, which are the
scenes: the decompiler also writes the pieces a scene stamps into itself, and
those are not pictures anybody asks for by number.

`gfx/chr/udg.chr` is the Amstrad edition's own character set, which is also
where its fills take their shading patterns from.

Two of the second part's drawings — 008 and 088 — are in the Spectrum edition
and not in the Amstrad's, where their table entries hold nothing but a return.
They are left out rather than borrowed.

## The layout numbers are the Amstrad's

The text is the Spectrum's and the screen is not: 40 columns where the Spectrum
has 42. So the handful of condacts that place things — `WINAT`, and the
`PRINTAT`s of the title and the menus — are the CPC edition's own, taken from
its own source, and they differ by a column or two:

| | +3 Extended | Amstrad CPC |
|---|---|---|
| the main window | `WINAT 0 5` | `WINAT 0 4` |
| the title's first line | `PRINTAT 11 21` | `PRINTAT 11 19` |
| the menus | `PRINTAT 8 5`, `PRINTAT 12 5` | `PRINTAT 8 4`, `PRINTAT 12 4` |

One column is the whole of it and it shows: the window is 40 wide and starts
where `WINAT` says, so at column 5 there are 35 columns left and the intro's
longest line is 36. It wrapped, and every line after it came out a word short.

## Why the disk does not carry all of them

Nothing compresses a picture on this machine. The interpreter reads one
straight into video memory, so a picture is the rows of a screen and no more
clever than that: fourteen character rows is sixteen kilobytes of a disk with a
hundred and seventy-eight on it, and nine of the twenty-eight fit.

`--lines` is where that is traded, and it is DAAD's own `IMGLINES` under
another name. These drawings are 111 rows tall and the title's frame is 112, so
112 is what it takes to carry one whole — and twelve character rows are 960
bytes of each band, which is the most that lets two bands share one of the 2K
records the file is read in, so 96 halves the size of every picture and fifteen
of the twenty-eight go on instead of seven. What it costs is the bottom two
character rows of every drawing, and these drawings use them: at 96 the frame
round the title has no bottom edge.

So the trade is only made here, on the disk that holds the pictures as they
are. The tape and the 128K disk compress them and carry all 112 rows.

Either way qpack names what would not fit rather than leaving it out quietly.

## The pictures on the tape are in memory

A tape cannot be asked in any order, so the pictures do not go on it: they go
**into memory with the game**, compressed, and the interpreter is pointed at
them.

What makes that fit is ZX0. A picture is a mode 1 screen with the rows the
adventure writes on blanked, and those come down to about 1300 bytes each — the
28 of the first part are 37K compressed against the 215K they take as they are.
What is free between the end of the game and the interpreter's own graphics
block is around fifteen, so thirteen of the twenty-eight ride along and qpack
names the rest.

The interpreter is pointed at them by two patches of three bytes each, at the
two places it calls the routine that opens a picture file: one asks whether a
picture is there and the other puts it on the screen. It already asks before it
draws, and it already takes no for an answer, which is what makes a picture that
did not fit come out as no picture rather than as a crash.

With no pictures at all the asking is taken out altogether, so the tape plays
through without ever stopping for a cassette. Left in, the adventure would run
perfectly well until the first `PICTURE` and then halt with "Press PLAY then any
key" in the middle of the Great Cavern.

The bank is read into memory by the loader, which is the one thing on the tape
that can, so a tape asked for pictures wants `--loading` as well. The Makefile
passes it.

## And on a 6128, all of them

```sh
qpack -f dsk --ram 128 --lines 96 --loading part1/gfx/loading.scr \
      -o dist/AO1-128 --name AO1 tmp/build-part1
```

`--ram 128` puts the bank in the four spare banks of a 6128 instead of in what
is left above the game. That is 64K the interpreter never touches — it writes
to no port but the PPI's and leaves the gate array's memory configuration alone
from beginning to end — so the loader pages each bank into the window at
`0x4000` as it reads it, and the routine that draws a picture pages it back for
as long as it takes to unpack.

All 28 pictures of the first part go in, and all 39 of the second. On a disk
that is worth as much as on a tape: a 128K disk is read once and the drive is
not touched again, where a 64K one opens a file every time the player walks
into a room with a drawing in it.

The price is the whole of the price: it will not run on a 464 or a 664.

