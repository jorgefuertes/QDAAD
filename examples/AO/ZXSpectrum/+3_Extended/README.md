# La Aventura Original for a Spectrum +3, with the 16-bit texts

The same two disks as [`../+3`](../+3/README.md) — same map, same puzzles, same
drawings, same walkthrough — carrying the **long descriptions of the Amiga, PC
and Atari ST editions** instead of the ones cut down for a tape.

A Spectrum of 1989 could not have them: the database has to fit in the memory
between the interpreter's workspace and the graphics stub, and on a tape there
was nothing to give. A +3 loads from disk and has the room, so this is the tree
where it gets them.

Everything about the disks themselves — the +3's own format, the 128 names, the
reserved track, why the disk is deliberately not bootable, which drawings are
left out and why, the loading screen — is in
[the README beside `+3`](../+3/README.md) and is unchanged here.

## Building

```sh
# Part 1
qdaad compile -i part1/game.sce -o build/p3x-part1 -f build -m zx -s plus3 -v 3 --part 1
qpack -f dsk -o AO1 --name AO1 build/p3x-part1     # AO1.dsk

# Part 2
qdaad compile -i part2/game.sce -o build/p3x-part2 -f build -m zx -s plus3 -v 3 --part 2
qpack -f dsk -o AO2 --name AO2 build/p3x-part2     # AO2.dsk
```

Or the `Makefile` here, which does it from this directory and needs nothing
installed — it runs the tools out of the repository:

```sh
make dist     # both disks, in dist/
make test     # both walkthroughs played, built in tmp/
make clean    # neither directory left behind
```

From the root, `make example-ao-p3x` builds the same two disks into `tmp/`.

## Where the texts come from

The 16-bit editions are **one text**: Amiga, PC and Atari ST hold byte-for-byte
the same `/LTX` and `/MTX`, so there is no choosing between them. What is here
is `work/decomp/ao/amiga`, which is also `work/decomp/ao/pc` and
`work/decomp/ao/atarist`.

They go in one for one. The numbering of all five text tables is the same in
every edition — 49 locations and 140 messages in part one, 109 and 163 in part
two — so this is a substitution and not a port: no entry moves, no process
changes, no `MESSAGE` or `SYSMESS` points anywhere new.

| | Locations rewritten | Messages rewritten |
|---|---|---|
| Part 1 | 19 of 49 | 23 of 140 |
| Part 2 | 35 of 109 | 51 of 163 |

Not one of them came out shorter than the tape's.

### What is not brought over

- **The object names.** There was nothing to bring: the 16-bit `/OTX` says "Una
  linterna encendida" and so does the tape's, and part two's table is identical
  down to the byte. Part one's differs in nine lines and in none of the words —
  the drawing beside each name is `#f#g` on the tape and `#g#f` there, the same
  two escapes in the other order.
- **The vocabulary, the connections, the object table and the processes.**
  Identical across every edition, byte for byte.
- **The system messages.** Two differ and neither is a long text. System message
  5 is empty on the tape and `"Vicente Misas "` on the 16-bit editions — a
  credit parked in an unused slot, not a description. And the 16-bit system
  message 49 of part one is missing the full stop ours has.
- **The compression table.** `tokens.sce` is already the same file in every
  edition, so the long texts compress against the table they were written for.

### The one thing that had to be rewritten on the way in

A `#g`…`#t` run switches to the alternate charset, where the capitals are a
title font — and that charset has no space in it. So a space inside a run comes
out as glyph 160 and not as a gap, and a run has to be one word:

```text
#gCUIDADO CON TUS OBJETOS#t          as the 16-bit editions write it
#gCUIDADO#t #gCON#t #gTUS#t #gOBJETOS#t   as it has to be written here
```

Eleven texts needed it. After the split the set of runs in this tree is exactly
the set in `+3`, which is the check that nothing else moved.

## It fits, with room left

The database is laid at 0x8400 and the graphics stub ends on the last byte of
RAM, 2095 bytes of it, so the stub starts at 0xF7D1 and the database has
**29649 bytes** between the two. That is the one limit a +3 build can meet, and
neither part comes near it:

| | Database | Of 29649 | Spare |
|---|---|---|---|
| Part 1 | 10637 → **12331** | 42% | 17318 |
| Part 2 | 14426 → **17379** | 59% | 12270 |

On the disk it costs two blocks in part one and three in part two, and the
directory does not move at all: the drawings are the same drawings, so the
names are the same names.

| | Names | Blocks used | Free |
|---|---|---|---|
| Part 1 | 33 of 128 | 90 of 175 | 85K |
| Part 2 | 46 of 128 | 119 of 175 | 56K |

So the whole of the 16-bit text fits, and there is no "the part that fits" to
talk about.

## Both parts are still won

`examples/AO/test/part1-extended.qtest` and `part2-extended.qtest` are the
walkthroughs of the `+3` tree with the same moves in the same order — the only
lines that differ are the `EXPECT`s that quote a text, which quote the long one
now. Part one is won and part two is played to the end.

The suite plays all four — `internal/qtest`,
`TestTheExampleScriptsPlay` — and `make qtest` runs them where they can be
watched.
