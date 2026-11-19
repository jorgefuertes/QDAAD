# La Aventura Original — Amstrad CPC

The +3 Extended tree with the Amstrad's own drawings: the same text, the
Amstrad's pictures, the Amstrad's font.

```sh
make dist    # every disk and every tape, in dist/
make dsk     # the 6128 disks only
make cdt     # the 464 tapes only
make cdt128  # the same tapes for a 128K machine, every picture in the banks
make test    # both walkthroughs played
```

or, without make:

```sh
qdaad compile -i part1/game.sce -o tmp/build-part1 -f build -m cpc -v 3 --part 1
qpack -f dsk --ram 128 --generic 27 --loading part1/gfx/loading.scr \
      -o dist/AO1 --name AO1 tmp/build-part1
qpack -f cdt --generic 27 --loading part1/gfx/loading.scr \
      -o dist/AO1 --name AO1 tmp/build-part1
```

The disk is a 180K DATA-format `.dsk` **for a 6128**, and the game starts with
`RUN"AO1"`. The tape is a `.cdt`, starts the same way, and runs on anything.

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

## Why the disk is a 6128 disk

Because the other one was not worth having, and it is worth saying what it was.

The interpreter opens a picture as a file and reads it straight into video
memory, with no compression anywhere in the path. So a drawing of fourteen
character rows is sixteen kilobytes of a disk with a hundred and seventy-eight,
and nine of the twenty-eight fit. `--lines` traded that — twelve rows instead
of fourteen fitted fifteen of them, at the price of the bottom of every drawing
and of the frame round the title entirely.

What the other nineteen cost was not nineteen drawings. `PICTURE` is a
condition in version 3: one that names a picture which is not there fails it,
and a failed condition **abandons the rest of the entry it is in** — which in
this adventure goes on to print the location's own text and to put the text
window back where the text goes. A disk that carries a third of the drawings
and mangles the layout at the rest is not a release.

So the disk is `--ram 128` and the pictures go in the spare banks, compressed,
where all 28 fit and the drive is never touched again. A 464 or a 664 loads
from tape, which is what those machines did in 1989 anyway. Asked for a disk
without the memory for it, `qpack` says so rather than writing one.

## The pictures on the tape are in memory

A tape cannot be asked in any order, so the pictures do not go on it: they go
**into memory with the game**, compressed, and the interpreter is pointed at
them.

What makes that fit is ZX0. A picture is the rows of a mode 1 screen the
adventure does not write over, compressed one band of video memory at a time,
and those come down to about 1900 bytes each — the 28 of the first part are 53K
against the 245K they take as they are. What is free between the end of the
game and the interpreter's own graphics block is around fifteen, so eight of
the twenty-eight ride along and qpack names the rest. A 6128 loading from tape
takes `--ram 128` and carries all of them.

Band by band and not one stream for the whole screen, which is what it was and
which cost a quarter less: a stream that covers all sixteen kilobytes carries
the rows below the picture as blanks, and unpacking it straight onto the display
writes those blanks over whatever is there. Where the adventure draws before it
prints, which is every ordinary location, nothing shows. Where it prints first
— the generic picture it puts up for a location with no drawing of its own — the
description came out black on black.

The interpreter is pointed at them by two patches of three bytes each, at the
two places it calls the routine that opens a picture file: one asks whether a
picture is there and the other puts it on the screen. It already asks before it
draws, and it already takes no for an answer, which is what makes a picture that
did not fit come out as no picture rather than as a crash.

## What the ones that did not fit show

`--generic 27`, which the Makefile passes, and it is not about looks.

In version 3 `PICTURE` is a **condition**, and one that names a picture which
is not there fails it — which abandons the rest of the entry it is in. Those
entries go on to do more than draw. At the grate of location 26 the entry is

```text
WINDOW 0 / PICTURE 46 / DISPLAY 0 / WINDOW 1 / MES 53 / MES 55 / MES 54
```

so a picture left off for want of memory takes the sentence about the grate
with it, and leaves the adventure printing in the graphics window: the prompt
comes out at the top of the screen and the description at the bottom.

So a picture that does not fit is not left out of the index. It is pointed at
the one the adventure already puts up for a location with no drawing of its own
— 27, the AD logotype among the trees — which is laid out first so that it is
never the one that does not fit. Every `PICTURE` succeeds, nothing is
abandoned, and the only thing missing is the drawing.

Without it, `qpack` says so rather than letting it pass:

```text
8 of the 28 pictures travel in memory with the game; 014, 015, … would not fit
and show nothing, which also costs whatever the rest of their entry would have
said: --generic names a picture to put up in their place
```

It is an Amstrad flag because the bank is the one place a picture is left out
at all: on the disk every one of them fits, and it is the tape above the game
that runs out of room.

With no pictures at all the asking is taken out altogether, so the tape plays
through without ever stopping for a cassette. Left in, the adventure would run
perfectly well until the first `PICTURE` and then halt with "Press PLAY then any
key" in the middle of the Great Cavern.

The bank is read into memory by the loader, which is the one thing on the tape
that can, so a tape asked for pictures wants `--loading` as well. The Makefile
passes it.

## And on a 6128, all of them

```sh
qpack -f dsk --ram 128 --generic 27 --loading part1/gfx/loading.scr \
      -o dist/AO1 --name AO1 tmp/build-part1
```

`--ram 128` puts the bank in the four spare banks of a 6128 instead of in what
is left above the game. That is 64K the interpreter never touches — it writes
to no port but the PPI's and leaves the gate array's memory configuration alone
from beginning to end — so the loader pages each bank into the window at
`0x4000` as it reads it, and the routine that draws a picture pages it back for
as long as it takes to unpack.

All 28 pictures of the first part go in, and all 39 of the second. The disk is
read once and the drive is not touched again, so nothing waits for it.

The price is the whole of the price: it will not run on a 464 or a 664.

`--ram 128` works on a tape as well, and for the same reason — a 6128 loading
from a cassette has the same four spare banks a 6128 with a drive has, so
`-f cdt --ram 128` carries every picture where a plain tape carries thirteen.
Either way it takes `--loading`, because the loader is the one thing that can
put them there, and a release asked for the banks without one is told so.

## Where the texts go

An adventure with `XMESSAGE`s carries them outside the database, and on this
machine that means files on the disk: the interpreter works out which one from
the offset the condact holds — the file is the offset divided by 2048 and the
position in it the remainder — and opens `00.XMB`, `01.XMB` and so on by name.
`qpack` cuts them and writes them.

A tape cannot: the interpreter opens those files through the firmware's
cassette vectors, and a text read from a position inside a file is not a thing
a tape can be asked for. A build with xmessages packaged as a `cdt` is refused
rather than written, which is what the 48K Spectrum tape does for the same
reason. Neither part of La Aventura Original has any.
