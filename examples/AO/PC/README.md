# La Aventura Original — PC/DOS

The +3 Extended tree measured for a PC, with the PC edition's **own** drawings.

```sh
make dist    # one directory per part, in dist/
make test    # both walkthroughs played
make clean
```

or, without make:

```sh
qdaad compile -i part1/game.sce -o tmp/build-part1 -f build -m pc -s vga256 -v 3 --part 1
qpack -f dir -o dist/AO1 --name AO1 tmp/build-part1
```

There is no container. A PC has a filesystem, so DAAD's release for it is the
files with the names the interpreter opens, and the directory `qpack` writes is
the release: mount it in whatever DOS you have and type `AO1`.

DOSBox is not in it. DAAD READY's PC build carries `dosbox.exe` and two SDL
DLLs, which is a Windows program for running a DOS program.

## Fifty-three columns

A PC in VGA256 is 320 by 200 pixels, which at eight by eight to the character
would be 40 columns — and is not. DAAD lays its text out in six-pixel columns
here, so the screen is **53 columns by 25 rows** where the Spectrum is 42 by 24
and the Amstrad CPC is 40 by 25 on the same pixels.

Every window number in this tree is measured for that and nothing else
differs: the text is the +3 Extended tree's text, the tables are its tables.
The main loop's windows go to the full 53, the text window under the picture to
25 rows less the picture's 14, and the boxes — the credits, the two menus in
part two — were moved rather than rewritten, because a `PRINTAT` is relative to
the window it sits in.

## The character set

This tree carries the adventure's own, `gfx/chr/udg.chr`, and that is not a
preference. DAAD's `#g` switches the printing routine into the top half of the
character set — `CharsetShift := 128`, ibmpc.pas:569 — and the top half is where
an adventure keeps its own drawn glyphs. La Aventura Original has 155 of them:
the frame around the credits, and two glyphs for every object it lists, so that
`Una botella#f#g&'#t` draws a bottle lying on its side out of glyphs 166 and
167.

Leaving it out was tried, to get the PC's own `PC.FNT` instead. It is the one
proportional font DAAD has — a width for each glyph, in the last 2304 bytes —
and it was the wrong trade twice over. The objects printed `Una botella` and
then a `ß` and an `É`, which is what those two codes are in a font that is not
this adventure's; and every line of text sat left inside its six-pixel column
because the glyphs no longer filled one.

So: the adventure's glyphs, six pixels each, and a screen of exactly 53
columns. `qpack` names it `DAAD.CHR`, which is the name the interpreter reads
that way (pcdaad.pas:236); `DAAD.FNT` is the proportional one and is what a
source declaring no character set still gets.

## The drawings

This is the one tree here whose pictures are not a port. La Aventura Original
**had** a PC release and these are its own drawings, out of its own archive;
the Amstrad tree borrows the CPC edition's and the PCW tree borrows these,
dithered down to one bit, because the PCW never had a release at all.

What changes is only the mode. The 1989 release was CGA mode 4 — four colours
on a 320 by 200 screen — and a version 3 interpreter draws VGA256 on the same
320 by 200, reading a `.MSD` built from a PCX. So the pixels come across
untouched, in the four colours of palette 1 at high intensity, which are the
four the decompiler already renders these drawings with.

`script/pcpcx` does the conversion:

```sh
go run ./script/pcpcx work/aventuras/La_Aventura_Original/PC/PART1.CGA part1/gfx/scr
go run ./script/pcpcx work/aventuras/La_Aventura_Original/PC/PART1.SCR part1/gfx
```

The second one is the cover, which arrives as a dump of CGA video memory rather
than as an archived picture, and which the packager turns into `DAAD.MSD` — the
screen the interpreter puts up before the adventure starts. The 1989 second
disks carry none and neither does this one.

**The drawings are 256 wide on a screen of 320, and they are centred.** Not by
padding them: the `.MSD` header carries a fixed position and the packager puts
the difference there, so a room is 256 across at x=32 rather than 320 across
with sixty-four columns of black down one side. Which is also why these PCX
files are not SCRMAKER's kind — it takes a whole screen and crops from the top
left, so it will not read one that is 256 wide.

It needs `work/`, so a fresh checkout does not run it: what the tree carries is
the PCX files it produced.

**Each drawing keeps its own height.** The rooms are 112 lines and the title is
176, and DAAD's own tooling crops every drawing in a build to one number —
`IMGLINES`, 96 — which either cuts the title's frame off or pads every room
with sixty-four blank rows that the interpreter then puts the text window
under. `qpack --lines` will do it the reference's way for anyone who wants the
reference's bytes; this tree does not ask for it.

## What the directories hold

| | Part 1 | Part 2 |
|---|---|---|
| Illustrations | 30 of 30 | 56 of 56 |
| Cover | yes | no, as in 1989 |
| Files | 35 | 60 |
| Size | 1.2 MiB | 2.0 MiB |

Megabytes, where a Spectrum tape is sixty kilobytes, and the reason is that
a `.MSD` is not compressed: sixteen bytes of geometry, the whole 768-byte
palette, and one byte a pixel. A 256 by 112 drawing is 28 KiB whatever is in
it. The format has a bit in its header for a run encoding and the reference
tool never sets it — and the interpreter never reads it — so there is nothing to
turn on. It went on a hard disk in 1991 and nobody minded.

Nothing is left behind, so none of the warnings a tape or a disk gets apply
here: a PC release has no size to run out of.

## What is not done

**VGA256 only.** DAAD READY builds no other PC — every one of its PC targets
calls the front end with `pc vga256` — and the rest are a different screen and
so a different conversion of every drawing. `TEXT`, `CGA`, `EGA` and plain
`VGA` are refused by name.

**Not the 640 by 400 screen.** `HIRES` in DAAD READY doubles the mode and
passes `-S` to the interpreter, which wants drawings at 640 by 192 and a cover
at 640 by 400. The drawings here are 256 wide; there is nothing to double them
from.
