# La Aventura Original — Amstrad PCW

The +3 Extended tree laid out for a machine twice as wide, with the PC
edition's drawings converted to its monochrome bitmaps.

```sh
make dist    # one disk per part, in dist/
make test    # both walkthroughs played
make clean
```

or, without make:

```sh
qdaad compile -i part1/game.sce -o tmp/build-part1 -f build -m pcw -v 3 --part 1
qpack -f dsk -o dist/AO1 --name AO1 tmp/build-part1
```

Neither disk boots. The player starts CP/M from their own system disk, puts one
of these in and types `DAAD`, which is what the PCW editions of 1989 asked for
and what DAAD READY's `PCW.BAT` still does. `qpack --boot` will put CP/M on the
disk and start the adventure by itself — it fits on the first part — but not on
the second, whose fifty-six illustrations leave no room for it, and two disks
used the same way are worth more than saving a word of typing.

## Ninety columns

A PCW is 720 by 256 pixels of monochrome, which at eight by eight to the
character is **90 columns by 32 rows** where the Spectrum is 42 by 24. Every
window number in this tree is measured for that and nothing else differs: the
text is the +3 Extended tree's text, the tables are its tables.

The boxes were moved rather than rewritten. A `PRINTAT` is relative to the
window it is in, so centring the credits meant moving window 5 and leaving the
nine `PRINTAT`s inside it alone. What did have to change is the text window,
which sits under the picture at row 14, and the handful of menus the second
part puts at the bottom of the screen.

## The drawings

La Aventura Original never had a PCW release, so these come from the PC's. CGA
mode 4 is four colours on a bitmap, so all that is lost is the colour — where
the Spectrum's drawings carry one ink per eight-by-eight cell and the CPC's are
vectors that version 3 cannot draw at all.

The arithmetic is Aventuras AD's own. Cozumel shipped on both machines and the
two archives can be held side by side: its PC illustrations are 256 by 104 and
its PCW ones 512 by 104, the same locations and the same count. They doubled
the width and left the height alone, which is what a PCW pixel being about half
as wide as it is tall asks for. So a room here is 512 by 112 and the title is
512 by 176, each picture keeping its own height.

`script/pcwpic` does the conversion:

```sh
go run ./script/pcwpic work/aventuras/La_aventura_original/PC/PART1.CGA part1/gfx/scr
go run ./script/pcwpic work/aventuras/La_aventura_original/PC/PART1.SCR part1/gfx
```

The second one is the cover, which on a PCW is a file of its own —
`PARTE000.PIC`, 640 by 200 in the machine's own cell order — and which the
interpreter puts up before anything else without being asked. The 1989 second
disks carry none, and neither does this one.

It needs `work/`, so a fresh checkout does not run it: what the tree carries is
the bitmaps it produced.

## What is not right

**The engraved text does not read.** Part two's cave has XYZZY cut into the
wall, black letters on a solid magenta plaque, and in the PC original it is
perfectly legible. Here it is not.

It is the dither, and the reason is worth writing down because the obvious fix
makes things worse. These drawings arrive dithered already — a CGA artist had
four colours and got more by alternating pixels — so the wall is a magenta and
black checkerboard while the plaque is solid magenta: the same colour, one
stippled and one not. Laying an ordered tile over that dithers a dither, and
what goes first is the thin stroke a letter is made of.

Resolving the source's stipple first and taking edges hard does make XYZZY
legible. It also turns the second part's title screen into a black rectangle,
because in these drawings the stipple is not noise over the picture, it **is**
the picture. See the comment on `dither` in `script/pcwpic`.

## What the disks hold

| | Part 1 | Part 2 |
|---|---|---|
| Illustrations | 30 of 30 | 56 of 56 |
| Cover | yes | no, as in 1989 |
| Of 173 KiB | 123 | 165 |

Both fit whole, and the packer would say so if they did not. What it will never
drop is a picture the adventure asks for **by number**: `PICTURE` is a
condition in version 3, so one that is missing abandons the rest of its entry —
and in part two the entry that draws picture 106 goes on to `GOTO 3`, which is
how the player leaves the opening room. A disk without it reads the access code
and then sits there for ever. The ones reached through a flag, `PICTURE @38`,
are the drawings proper and those can go: the generic stands in and the
adventure carries on.
