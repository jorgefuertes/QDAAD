# La Aventura Original for a 48K Spectrum

Two tapes, one per part. Each carries its BASIC loader, the loading screen, the
interpreter, the database, the pictures and the stub, all in one stream.

What follows assumes `qdaad` and `qpack` are on the PATH. For where the sources
come from and what they carry corrected by hand, see
[the README above](../../README.md).

## Building

```sh
# Part 1
qdaad compile -i part1/game.sce -o build/48-part1 -f build -m zx -s 48k -v 3 --part 1
qpack -f tap -o AO1 --name AO1 build/48-part1     # AO1.tap
qpack -f tzx -o AO1 --name AO1 build/48-part1     # AO1.tzx

# Part 2
qdaad compile -i part2/game.sce -o build/48-part2 -f build -m zx -s 48k -v 3 --part 2
qpack -f tap -o AO2 --name AO2 build/48-part2     # AO2.tap
qpack -f tzx -o AO2 --name AO2 build/48-part2     # AO2.tzx
```

The `tap` and the `tzx` carry the same blocks: the second is the first with a
signature and a one-second pause after each block. It loads the same and weighs
46 bytes more.

## What the loader does

```text
10 CLEAR VAL "24575"
20 BORDER 0: PAPER 0: INK 7: CLS
30 LOAD ""CODE                                      the screen, to 16384
40 POKE PEEK 23633+256*PEEK 23634,VAL "111"         quiet the ROM
50 LOAD ""CODE:LOAD ""CODE:LOAD ""CODE:LOAD ""CODE  interpreter, database, stub, pictures
60 RANDOMIZE USR VAL "24576"                        and away
```

The screen goes to video memory, so it paints itself as it arrives and needs
neither a routine nor a copy. The loader of 1989 put it at 30000 and blitted it
with a `RANDOMIZE USR 30000`; this is the same thing with fewer steps. The
`POKE` is the usual trick to stop the ROM writing "Bytes:" over the picture,
with the address read out of `CURCHL` rather than written down, which is how
the original does it in its line 200.

Loading the last block starts the adventure, with no key in between.

## The pictures

A 48K tape has room for neither part one's 41 nor part two's 69: about ten fit.
So each part carries only what the game needs to make sense, and the generic
picture covers everything else.

| Part 1 | | Part 2 | |
|---|---|---|---|
| `003` | the brick cabin in the wood | `001` | the title screen, which is also the first location |
| `027` | the generic one | `095` | the generic one |
| `029` | the gravestone, on dying | `092` | the gravestone |
| `044` | the frame around the credits | | |

The generic picture standing in for the rest is not a shortcut of ours: it is
what the adventure does, and it decides with flag 40 — "this location has a
drawing" — which the loop puts back. To carry more, drop their `.scr` into
`partN/gfx/scr/` and package again: `qpack` says by itself which ones did not
fit, and never cuts one short.

## Part two asks for a password

That is the original's doing: it opens with "¿Clave de acceso?", and the word
was what finishing part one gave you.
