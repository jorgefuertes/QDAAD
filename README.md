# QDAAD

A Go reimplementation of DAAD, the adventure writing system.

*[Léeme en español](README_ES.md)*

Four tools, and between them a round trip: **read an adventure off the tape it
shipped on in 1989, and write a tape you can play**.

| | |
|---|---|
| `qundaad` | reads a DAAD database — or the disk or tape image holding it — and gives back the source that produced it |
| `qdaad` | compiles that source back into a database |
| `qpack` | packages a build into the tape, disk or files a machine loads |
| `qrun` | runs a database here, without an emulator |

The round trip is not a claim, it is a target in the Makefile.
`examples/AO/` holds La Aventura Original decompiled, converted from version 1
to version 3, and corrected by hand where the conversion cannot yet manage;
`make example-ao` builds it into two 48K tapes and two +3 disks. Part one of
the +3 disk has been played from start to finish in ZEsarUX, and part two as
far as anybody has got. See [its README](examples/AO/README.md).

## What is checked, and against what

Three of these compare our output with the reference tooling's, byte for byte.
The reference is DAAD Reborn by Uto, which had nothing to do with writing this.

| `make` | | |
|---|---|---|
| `decomp-all` | decompiles every edition and holds them against one another | **38 pairs** |
| `recompile-all` | hands every decompilation to the reference compiler | **66 accepted** |
| `compile-all` | converts every edition to version 3 and compiles it back | **66 compiled, 0 failed** |
| `compare-drf` | reads every source with both front ends and compares what each understood | **70 agree, 0 differ** |
| `compare-drb` | links every source with both back ends and compares the databases | **198 agree, 0 differ** |
| `compare-daadmaker` | packs every source into a 48K tape with both packagers | **63 agree, 0 differ** |

`compare-drb` covers three targets per source so that the byte order and the
padding are both exercised. What it skips is what the reference will not link
at all; what `compare-daadmaker` skips is sources in languages its compiler
does not speak, and ones too big for a 48K Spectrum.

## qundaad, the decompiler

```sh
go build -o dist/qundaad cmd/qundaad/qundaad.go

qundaad decompile --input PART1.DDB              --output src/
qundaad decompile --input ORIGINAL.ADF           --output src/
qundaad decompile --input "Aventura Original.dsk" --output src/

# only what can be read, without copies of the original binaries
qundaad decompile --input ORIGINAL.ST --output src/ --no-binaries
```

The input can be the database on its own or **the image of the original disk or
tape**. An image is opened, walked, and every database inside it decompiled into
a directory of its own.

The output is UTF-8, readable and editable today, split one file per section and
joined with `#include` from `game.sce` — which is how the sources of the time
were organised: the 1991 manual warns about the dangers of "including a TOK file
which contains an extra `/TOK`", a warning that only makes sense if splitting
was the normal practice.

**It is source a compiler accepts**, not a listing — and not only ours:
`recompile-all` hands every decompilation to DAAD Reborn Compiler 0.40, and all
66 are accepted once the encoding is put back to ISO-8859-1 and the `/TOK`
section, which version 3 dropped, is left out.

Three of them are accepted and should not be trusted: the process tables of
Cozumel on the Spectrum still cannot be read, and their source says so at the
top. Compiling would give a game, but not that one.

Everything that travelled with the database goes beside the source in `gfx/`,
sorted one directory to the kind of file: `png/` for the pictures, `plot/` for
the drawing listings of the eight-bit editions, `chr/` for the character sets,
and `raw/` for the bytes as they shipped. So each thing is kept **as the binary
it came as and as a PNG**, and the archives of illustrations are cut into their
separate drawings as well, with an index naming which locations use each and in
what palette. `--no-binaries` drops `raw/` and keeps only the conversions, which
halves the space.

### What it works out instead of assuming

A `.DDB` declares almost nothing of what is needed to read it. `qundaad` does not
trust tables indexed by platform — which is how the reference tooling comes to
report a PC database as *little-endian* when it is not — and takes it from the
data itself:

- **Byte order and header size**, by trying all four combinations and keeping the
  one that makes the header describe itself. All four turn up across these five
  adventures.
- **The address it was loaded at**, when the database is not a file but is linked
  into a program or laid straight onto a disk. Its pointers are absolute
  addresses, and the vocabulary gives away where it was all loaded — but only
  *near enough* after the header, since some builds left padding between the
  two, six bytes on the Amstrad PCW and twenty-six in Los Templos Sagrados. So
  every width of padding that has a word at the end of it is tried, and the
  reading that parses decides. That is what turns up 0x100 for the PCW, which
  is where CP/M loads a program.
- **Whether there is a compression table**, which is optional: the CGA build of
  El Jabato was compiled without one.

### The media it opens

`internal/media` reads the container of each machine, because a database is
hardly ever a contiguous run of bytes on the original medium:

| Medium | |
|---|---|
| Amiga `.ADF` | OFS (with a header in every data block) and FFS |
| Atari ST and MSX `.ST`, `.DSK` | FAT12; the MSX files are BLOAD images |
| Commodore `.d64` | CBM DOS, linked sectors |
| Amstrad and Spectrum `.DSK` | CPCEMU, both the original and extended variants |
| Amstrad CPC, PCW and Spectrum +3 | CP/M, where the disk carries a directory |
| `.TZX` and `.CDT` tapes | all 25 block types of the format |
| Commodore `.TAP` | no: it holds the pulses, not the bytes |

The `.DSK` disks come **two ways, and the container never says which**. Some are
ordinary CP/M volumes: the Amstrad PCW ones name their files outright, and so,
it turns out, does the Spectrum +3 disk of Cozumel. Others were formatted for
the game's own loader and hold nothing but a run of sectors. Reading the
directory is the test — and it has to be a strict one, since a disk of program
and picture data has sectors that pass for a directory if you let them.

Where there is none, the databases are found by signature among the sectors and
made to prove they are one — a header that describes itself, a vocabulary whose
words are words, text tables and connection lists that read, and prose that
reads like prose — before being accepted.

## The pictures and the character sets

A character set is 2176 bytes: a **128-byte AMSDOS header and 256 glyphs of eight
rows**. That the Amstrad header is still on them in the PC edition says where the
tooling came from — Aventuras AD worked on the CPC and the PCW. The letters are
drawn in six of the eight columns; the glyphs from 128 up are not letters but
**shading patterns**, and those do use all eight.

They also confirm the DAAD character set from the bitmaps themselves rather than
from other people's tables: 16 = ª, 17 = ¡, 18 = ¿, 19 = «, 20 = », 21 to 25 =
á é í ó ú, 26 = ñ, 27 = Ñ, 28 = ç, 29 = Ç, 30 = ü, 31 = Ü.

The **loading screens** are drawn four ways: CGA mode 4 with its interleaved
banks, EGA mode 0Dh in four planes, and 68000 Degas — which on the Atari
interleaves the planes a word at a time and on the Amiga keeps each whole,
something the file does not say and has to be taken from the disk it came off.

There are **three compression schemes**, one per generation, and none is a known
format. The PC one came out of the compressor itself: `DMG.EXE`, the graphics
manager that built these archives, still ships with the 1991 release, and the
routine at 0x3b4f of it counts how often each byte value occurs in a picture,
keeps the four commonest, and writes the picture a byte at a time — except that
a byte among those four is followed by a count of how many times it repeats. How
many of the four are treated that way is chosen by trying all four and keeping
whichever came out smallest. Read back, 189 of the 191 compressed pictures
unpack to exactly the size their header implies, having eaten their whole
stream.

The other two are the 68000 ones, and neither yields to any standard scheme: a pixel is four bits, and the header carries **a sixteen-bit
mask saying which colours are followed by a run length**. Because the meaning of
the next four bits depends on the colour just read, no standard scheme fits. It
came from reading the interpreters with `objdump -m m68k` — the routine at
0x20aa of the Atari one for La Aventura Original, and 0x269a for Los Templos
Sagrados.

There are two generations of archive, and nothing in the file says which it is.
The later adventures widened the slot from 44 bytes to 48 and changed where the
pixels are gathered from — a nibble of a longword instead of a bit from each of
four bytes — and their PC editions hold the same archive with the bytes the
other way round. Each shape is tried and the one that adds up is kept: an
archive lays its first picture exactly where its table of slots ends, and only
the right reading puts it there.

**1798 pictures** are converted as things stand: character sets, loading screens
and illustrations, on every machine whose databases can be read.

## qdaad and qpack, the other way round

```sh
go build -o dist/qdaad ./cmd/qdaad
go build -o dist/qpack ./cmd/qpack

qdaad compile -i game.sce -o build/ -f build -m zx -s 48k -v 3 --part 1
qpack -f tap -o GAME --name GAME build/      # GAME.tap
```

The two were one command and were split because they are two jobs. Compiling is
about one source and gives a database; packaging is about a **release**, which
is about a machine. `qpack` writes `tap`, `tzx`, `dsk`, `dir` — everything the
machine needs as loose files — and `qrun`, our own container.

What goes into a release beyond the database is declared by the source, so the
same build directory packages for any of them:

```text
; qdaad: charset=gfx/chr/udg.chr    the font, which version 3 does not carry
; qdaad: images=gfx/scr             a directory of Spectrum screens
; qdaad: loading=gfx/loading.scr    the picture shown while the rest loads
```

A **48K tape** is one stream: a BASIC loader that moves RAMTOP out of the way,
then the loading screen, the interpreter, the database, the graphics stub and
as many pictures as the memory left will hold. The loader is written rather
than borrowed, and it is the same bytes daadmaker writes, down to a leading
space on one line — which is what makes `compare-daadmaker` mean anything. A
`tzx` is the same blocks wrapped, 46 bytes heavier.

A **+3 disk** is the files loose, and it is written in the +3's own format
rather than the CPC's DATA one that a CPC disk still gets. The difference is
sixteen bytes in the first sector of track 0 — the disk specification — and the
reason is the byte at +7 of it: how many blocks the directory takes. A DATA
disk has no specification, so the answer is 64 names and there is no arguing
with it; saying four blocks instead of two makes it 128, for two kilobytes of a
disk that has eighty spare. A picture is a file and a file is a name, so for an
adventure with a drawing per location, names are what runs out first.

The specification is La Aventura Original's own, read off its disk, with that
one byte changed. Verified on a +3: the menu loads the disk, a `CAT` lists
every name, and the free space it reports is the arithmetic of the fields that
moved and not of their defaults.

## qrun, the interpreter

```sh
go build -o dist/qrun ./cmd/qrun
qrun -i GAME.qrun
```

It runs a version 3 database here, on the terminal, which is how a change to
the compiler gets tried without an emulator in the way. It is not a Spectrum:
there are no pictures and no character set, so what it is good for is the
parser, the flags and the process tables. `TIME` and its timeout work, because
a question with a clock on it is a thing adventures do and a thing that is
tedious to test by hand.

### Testing a whole adventure: `.qtest`

A walkthrough is not code. It is a list of what to type and what should come
back, written while playing, so it lives in a file rather than in a Go test:

```sh
qrun -i tmp/AO1.ddb -T examples/AO/test/part1.qtest
```

```qtest
TEST "the forest, and the bottle"
SEED 1989

STEP "the title screen, and then the forest"
  EXPECT  "?"                 ; it asks for a key before showing anything
  KEY     SPACE
  EXPECT  "AVENTURA ORIGINAL" ; the credits, and it asks again
  KEY     SPACE
  EXPECT  "LA RESPUESTA DEPENDE DE TU VALOR"
  KEY     SPACE
  EXPECT  "Estás en el bosque"
  ASSERT  AT = 3
END

STEP "the bottle, four rooms down the valley"
  INPUT   S
  INPUT   COGER BOTELLA
  EXPECT  "Has cogido la botella"
  EXPECT  NOT "Aquí no hay de eso"
  ASSERT  CARRIED 4
END

STEP "the time passes, and does not come back"
  WAIT    50              ; fifty seconds of saying nothing, for real
  EXPECT  "El tiempo pasa"
END
```

A caret announces each step as it opens and the verdict lands at the end of
that line when its `END` is reached, which is the proof that every order in it
held:

```text
╭─────────────────────────────────────╮
│ La Aventura Original, part one, won │
╰─────────────────────────────────────╯
▶ the title screen, and then the forest     OK
▶ the bottle, down the valley and back      OK
▶ the boots, east and south                FAIL
  examples/AO/test/part1.qtest:63:3 EXPECT "Has cogido la pareja de botas"
  it did not say that in 5 seconds
  what it said after INPUT COGER BOTAS:
    > ¡Aquí no hay de eso!

▶ 2 steps passed
▶ 1 step failed
```

A step that does not hold stops the run and `qrun` exits non-zero. One order is
one command — `INPUT` types its words and presses return — and `KEY` is the one
that types without sending, for the single place that wants it: a yes-or-no
question answered with one letter. The whole language is `EXPECT`,
`EXPECT NOT`, `KEY`, `INPUT`, `WAIT`, `DEBUG`, `ASSERT`, `TRACE`, `TIMEOUT`,
`SEED`, `INCLUDE` and `GAME` — `qrun --help` has the reference, and
[docs/qtest.md](docs/qtest.md) the whole of it.

Nothing in it changes what the adventure does, and the pauses are where that
shows: the adventure asked for a key, so the script presses one. Deciding that
a pause does not count would be the harness changing the game, and how many
pauses there are is part of what a test of it is for.

The seconds in it are seconds, and a step that waits waits. That is how "El
tiempo pasa... ¡Y no vuelve!" and the dwarf walking into the room become
testable: the script says nothing for long enough that the adventure's own
`TIME` runs out, and the runtime runs that turn for its own reasons. Reporting
a timeout that had not happened would cost nothing and would make the adventure
run a turn it never would have — observing the run by altering it. A failing
`EXPECT` pays its seconds; a passing one costs nothing, because what it is
looking for was said before the adventure asked for an order.

`examples/AO/test` is the first part of La Aventura Original won — 84 commands,
a conversation, a four-stage puzzle chain and the access code it prints at the
end — and the second part entered. `make qtest` builds the databases and plays
them; the suite plays them too, against the same walkthrough the Go tests use,
so if the two ever disagree one of them is lying.

## What it has been tested with

The five Aventuras AD adventures, in every edition that could be found: **50
media files, yielding 66 databases**.

| Adventure | PC | Amiga | Atari ST | Amstrad CPC | ZX Spectrum | PCW | MSX | C64 |
|-----------|----|-------|----------|-------------|-------------|-----|-----|-----|
| La Aventura Original | yes | yes | yes | yes | yes | — | — | no |
| El Jabato | yes | yes | yes | no | yes | — | no | no |
| Cozumel | yes | yes | yes | yes | yes | yes | no | no |
| Chichén Itzá | yes | yes | yes | yes | yes | — | **yes** | no |
| Los templos sagrados | yes | yes | yes | yes | yes | yes | no | no |

A cell says whether the adventure reads on that machine, both of its parts, not
whether every file shipped for it does. An em dash means no edition for that
machine was found, which is why the PCW column is mostly empty: only two of the
five were ever sold for it.

Three gaps are worth spelling out. Both of El Jabato's are on that row: its
second Atari ST disk holds a single file, the picture archive of part two, so
there is no database on it to read, and its Spectrum tapes do not read at all,
though the disk for the same machine gives both parts. The third is the whole
**Commodore 64** column, and it is one problem rather than five: every `.d64`
holds a single self-starting program with its payload packed, at 6.96 to 7.86
bits to the byte against the 6.71 of a database in the clear. Nothing can be
found in them until the packer is identified and undone.

The **MSX** column has one yes and it was the hardest earned. Those disks are
ordinary MS-DOS ones, but no file on them holds a database: every piece loads
into the same buffer and copies itself down into the RAM the game runs in, so
what is read is the memory those copies build. See `msx.go`. Cozumel and El
Jabato do not come out yet — Cozumel divides its memory another way, and the
Jabato disk gives up no files at all — and the `.CAS` tapes of all five need a
reader that does not exist.

### How the result is checked

`make decomp-all` decompiles the five adventures and holds the editions against
one another: **38 pairs**. Fifteen have to give the same source whole, nineteen
have to agree on the text and the data, two have to trace the same drawings, and
two have to describe the same world in other words — the MSX edition of Chichén
Itzá against the PC one, whose prose was rewritten for a forty-column screen but
whose objects, object text and connections did not change.

It is a strong check because the binaries compared have nothing in common —
different byte order, different size, different offsets throughout — and still
have to converge on the same text. Break the byte order deduction, the token
table or the pointer arithmetic and one of them stops matching.

The two comparisons that say the most:

- **The tape against the disk.** Two containers with nothing whatever in common
  — a stream of blocks written for the ear, against a table of sectors — giving
  the same text, byte for byte.
- **EGA against CGA in El Jabato.** One was compiled with a compression table and
  the other with none at all, and every line of prose comes out identical. That
  says rather more about token expansion than comparing two identical tables.
- **The Amstrad PCW against the PC in Los Templos Sagrados.** A Z80 build for
  CP/M, laid at 0x100 with padding after its header, against a PC one of the
  other byte order: all seven shared sections come out identical, both parts.

Things about the adventures have come out along the way: **the Amiga shipped the
database compiled for the Atari**, without recompiling it, in both La Aventura
Original and El Jabato.

### What does not read, and why

Nearly everything that fails shares one cause, and it is **not copy protection**.
The custom disk format partly is, but that much is already got past. What blocks
it is that **the loader keeps the database in pieces** and assembles it in RAM,
so on the medium it is never contiguous.

The Amstrad disks say as much outright. They do carry a CP/M directory, and what
it names is `JABATO.B00`, `.B01` and `.B02`, or `TEMPLOS1.B00` through `.B02`:
the loader's banks, numbered. The files are there to read and the database is
spread across them.

The proof is the MSX edition of El Jabato: its header, its 288 vocabulary words —
the same as the PC edition — and its 128 tokens all decode perfectly. With
encryption or compression not one of the three would read; only the text is
somewhere else.

The exception is the Spectrum tapes of El Jabato, which load in 128-byte chunks
and do transform the data. Even there, the disk for the same machine reads
without trouble.

Still to do on the databases: work out the load map of those loaders — it would
unblock MSX, Amstrad CPC and Commodore 64 in one go — and read the MSX `.CAS`
and Commodore `.T64` formats.

Nothing is left of the pictures on the machines whose databases read. Every
illustration of every edition that can be reached comes out.
