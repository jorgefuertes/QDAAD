# qDAAD

A Go reimplementation of DAAD, the adventure writing system.

*[Léeme en español](README_ES.md)*

The project has two halves. `qdaad` will compile `.SCE` sources, along with a
more modern format of its own, into `.DDB` databases, and is under construction.
**`qundaad` already works**: it reads a DAAD database and gives back the source
that produced it.

## qundaad, the decompiler

```sh
go build -o dist/qundaad cmd/qundaad/qundaad.go

qundaad decompile --input PART1.DDB              --output src/
qundaad decompile --input ORIGINAL.ADF           --output src/
qundaad decompile --input "Aventura Original.dsk" --output src/
```

The input can be the database on its own or **the image of the original disk or
tape**. An image is opened, walked, and every database inside it decompiled into
a directory of its own.

The output is UTF-8, readable and editable today, split one file per section and
joined with `#INCLUDE` from `game.sce` — which is how the sources of the time
were organised: the 1991 manual warns about the dangers of "including a TOK file
which contains an extra `/TOK`", a warning that only makes sense if splitting
was the normal practice.

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
  addresses, and the vocabulary, which sits right after the header, gives away
  where it was all loaded.
- **Whether there is a compression table**, which is optional: the CGA build of
  El Jabato was compiled without one.

### The media it opens

`internal/media` reads the container of each machine, because a database is
hardly ever a contiguous run of bytes on the original medium:

| Medium | |
|---|---|
| Amiga `.ADF` | OFS (with a header in every data block) and FFS |
| Atari ST and MSX `.ST`, `.DSK` | FAT12 |
| Commodore `.d64` | CBM DOS, linked sectors |
| Amstrad and Spectrum `.DSK` | CPCEMU, both the original and extended variants |
| `.TZX` and `.CDT` tapes | all 25 block types of the format |
| Commodore `.TAP` | no: it holds the pulses, not the bytes |

Several of these disks **carry no filesystem**: they were formatted for the
game's own loader. There is no directory to walk, so the databases are found by
signature among the sectors and made to prove they are one — a header that
describes itself, text tables and connection lists that read, and prose that
reads like prose — before being accepted.

## What it has been tested with

The five Aventuras AD adventures, in every edition that could be found: **81
media files, yielding 49 databases**.

| Adventure | PC | Amiga | Atari ST | Amstrad CPC | ZX Spectrum | C64 | MSX | PCW |
|-----------|----|-------|----------|-------------|-------------|-----|-----|-----|
| La Aventura Original | yes | yes | yes | yes | yes | no | — | — |
| El Jabato | yes | yes | 1/2 | no | 1/3 | no | no | — |
| Cozumel | yes | yes | yes | yes | yes | no | no | no |
| Chichén Itzá | yes | yes | yes | no | no | no | no | — |
| Los templos sagrados | yes | yes | yes | no | no | no | no | no |

("1/2" for the Atari ST of El Jabato is its second disk, which holds only
graphics.)

### How the result is checked

`make decomp-check` decompiles the five adventures and holds the editions
against one another: **29 pairs**, twelve that have to give the same source
whole and seventeen that have to agree on the text and the data.

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

Things about the adventures have come out along the way: **the Amiga shipped the
database compiled for the Atari**, without recompiling it, in both La Aventura
Original and El Jabato.

### What does not read, and why

Nearly everything that fails shares one cause, and it is **not copy protection**.
The custom disk format partly is, but that much is already got past. What blocks
it is that **the loader keeps the database in pieces** and assembles it in RAM,
so on the medium it is never contiguous.

The proof is the MSX edition of El Jabato: its header, its 288 vocabulary words —
the same as the PC edition — and its 128 tokens all decode perfectly. With
encryption or compression not one of the three would read; only the text is
somewhere else.

The exception is the Spectrum tapes of El Jabato, which load in 128-byte chunks
and do transform the data. Even there, the disk for the same machine reads
without trouble.

Still to do: work out the load map of those loaders — it would unblock MSX,
Amstrad CPC and Commodore 64 in one go — and read the MSX `.CAS`, Commodore
`.T64` and PCW disk formats.

## Layout

```
cmd/qundaad/       the decompiler and its CLI
cmd/qdaad/         the compiler, under construction
internal/ddb/      the data model: vocabulary, objects, messages, processes
internal/media/    reading the disks and tapes of each machine
docs/              research on the format, and the manual
work/              reference material, not part of the program
```

## Development

```sh
make test           # go vet and the full suite
make lint           # gofumpt, staticcheck, golangci-lint, govulncheck
make decomp-check   # decompiles the five adventures and compares them
make dead           # unused code
```
