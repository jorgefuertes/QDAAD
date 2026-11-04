# The `.qtest` language

A walkthrough is not code. It is a list of what to type and what should come
back, written while playing, and the place for it is a file somebody can edit
with the game open in the other window. `qrun --test` reads that file.

```sh
qdaad compile -i examples/AO/ZXSpectrum/+3/part1/game.sce -m pc -s vga -f ddb -o tmp/AO1
qrun -i tmp/AO1.ddb -T examples/AO/test/part1.qtest
```

It exits zero if every step held, and non-zero with the line that did not:

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

    >
  and before that, still unmatched:
    > El cañón se abre en lo que fué antes una cascada.

    >

▶ 2 steps passed
▶ 1 step failed
```

The block that matters is the second one: what the adventure answered to the
last order the script gave, with the order named above it. That is where the
phrase for an `EXPECT` comes from when a script is being written — everything
still unmatched is the leftovers of however many commands ago, which is the
block below it and the one to read when a step failed because its `EXPECT`s are
in the wrong order. Runs of blank lines are squeezed in both, because an
adventure pads with newlines wherever its author felt like it.

A caret announces each step as it opens, so a run that hangs has already said
which step it was in, and the verdict lands at the end of that line when the
`END` is reached — green for `OK`, red for `FAIL`, both the same width so they
read as one column. Anything said inside a step, a `DEBUG` or the adventure's
own words under `--show`, breaks that line, and the badge then stands on its
own at the margin.

## The clock, which is the one thing here that is not obvious

An adventure that has set `TIME` gives the player a few seconds to answer, and
when they run out the tables run on without an order — "El tiempo pasa... ¡Y no
vuelve!", or the dwarf walking into the room. That is half of what an adventure
does and it has to be testable, so **the seconds are seconds**: a step that
waits waits, the same as a player who sits still.

Whoever runs out first decides.

```qtest
EXPECT "El enano entra"       ; five seconds, which is what it takes when
                              ; nothing says otherwise. The adventure allows
                              ; 47, so this gives up first: the step fails
                              ; and the game has not moved an inch

EXPECT "El enano entra", 60   ; now the adventure's own timeout fires while
                              ; the step is still waiting. The runtime runs
                              ; its turn, for its own reasons, and what it
                              ; prints may be the very thing being waited for
```

What is never done is to report a timeout that did not happen. It would cost
nothing and it would be a lie with consequences: the adventure would run a turn
it would never have run and say something a player would never have seen, which
is observing a run by altering it.

So a failing `EXPECT` costs its seconds, and a passing one costs nothing — the
text it wants is already in the buffer, because the machine said everything it
had to say before it asked for an order. A green run is as fast as the
interpreter.

When the adventure has no clock running on the question — which includes the
yes-or-no questions `QUIT` and `END` ask — there is nothing to wait for at all:
it is holding out for an order and will say nothing more until it gets one. An
`EXPECT` then fails at once and says so, rather than spending real seconds to
learn nothing.

## How a script and a machine take turns

The machine is in charge and the script is what it pumps. Every step runs
inside one of the three calls the machine makes of a console — a line in, a
line in with a clock on it, a pause — plus one last pass once the game is over.
There are no threads, so a run is repeatable.

A **pause waits**, because the adventure asked it to. A script with nothing to
press fails there, with the line it got to, rather than being let through:
letting it through would be the harness deciding that this pause does not
count, and a pause is the adventure's own behaviour — how many there are and
where they fall is part of what a test of it is for. A walkthrough that stops
working the day somebody adds one is a walkthrough that noticed.

So a script reads like a session. La Aventura Original asks for three keys
before the game begins, and this is what answering them looks like:

```qtest
STEP "the title screen, and then the forest"
  EXPECT  "?"                                ; it asks before showing anything
  KEY     SPACE
  EXPECT  "AVENTURA ORIGINAL"                ; the credits
  KEY     SPACE
  EXPECT  "LA RESPUESTA DEPENDE DE TU VALOR" ; what the game is about
  KEY     SPACE
  EXPECT  "Estás en el bosque"
END
```

## Nothing here changes what the adventure does

The machine is not told that it is being tested and has no way of finding out.
What a script hands it is a line, a keypress, or a player who said nothing —
which is all a person hands it too.

That is the line the language is drawn along, and it is worth being exact about
where the clock falls on it. When an `EXPECT` with seconds on it lets the time
run out, **the runtime is the one running the turn**: it set the `TIME`, its own
timeout expired, and the console did nothing but report truthfully that nobody
answered within the limit the adventure itself chose. A person sitting still for
forty-seven seconds produces the same thing. What the harness never does is
give an order that nobody wrote.

## Writing one

One order per line, keywords in whatever case you like, `;` starts a comment —
the same as in a DSF. A text argument is the rest of the line, or what is
between quotes if it starts with one; quotes are what to reach for when the
text has a comma, or edge spaces that matter. Files are UTF-8. A quoted text
reads `\n`, `\t`, `\"` and `\\`, and leaves every other backslash exactly as it
found it — a line break is worth having, because a `TEST` or a `STEP` whose name
is two lines makes a better heading than one long one.

One order is also one command: `INPUT coger tortilla` types the words and
presses return. `KEY` is the one that types without sending, and the place that
wants it is a question answered with a single letter — `KEY S` then
`KEY RETURN`. A `KEY RETURN` that nothing is waiting for is not refused: it
hands over an empty line, the adventure answers it with its "I do not
understand", and that answer in the transcript is how it shows.

```qtest
; La Aventura Original, part two
TEST    "the second part, entered"
GAME    "../../../tmp/AO2.ddb"   ; optional; -i wins over it
SEED    1989

INCLUDE "prologue.qtest"         ; the title screen and the access code

STEP    "the first room"
  EXPECT  "Los rayos del sol", 2
  ASSERT  AT = 1
  ASSERT  PICTURE = 1
END

STEP    "what the first part handed over"
  INPUT   i
  EXPECT  "Una llave"
  EXPECT  NOT "No tienes nada"
  ASSERT  CARRIED 12
END

STEP    "the time passes, and does not come back"
  WAIT    50
  EXPECT  "El tiempo pasa"
  DEBUG   FLAGS SET
END

STEP    "FIN ends it"
  INPUT   fin
  EXPECT  "seguro"
  INPUT   s
  EXPECT  "otra partidita"
  INPUT   n
  ASSERT  ENDED
END
```

The indentation inside a `STEP` is nobody's business but the author's; the
language does not look at it.

### The step is the unit of the report

`STEP` opens one and `END` closes it, and every `STEP` has to have its `END`.
That is checked when the file is read, not when it runs: an unclosed `STEP` or a
stray `END` is reported with its line before the adventure is even started.
They do not nest, and they have to balance inside one file, so an `INCLUDE`
brings whole steps and never half of one.

What prints is `END`. Reaching it means everything inside the step held, so
that is where `OK` goes. A failure stops the run — a walkthrough that is one
answer behind answers every remaining step in the wrong room, and reports
nonsense in ten places instead of the truth in one.

## The orders

| Order | What it does |
|---|---|
| `EXPECT <text>[, <seconds>]` | Wait for the text. Matched case-insensitively against what has been said and not matched yet; on a hit the buffer is eaten **up to the end of the match**, which is what makes the order things are said in part of the test. Any run of space in the text matches any run of space in what was said, because the machine's line breaks are wherever the database put them. The seconds are how long it is prepared to wait, really, and are **5** if not given — see the clock above. |
| `EXPECT NOT <text>` | Fails if the text is in what has not been matched. Eats nothing, and takes no seconds: it is about what has been said already. |
| `INPUT <text>` | One command: types the text **and** presses return. Fails if there is nowhere to type, which is the game having ended. |
| `KEY <key>` | `RETURN`/`ENTER`, `SPACE`, `ESC`, or one character. At a pause it is the keypress the adventure is waiting for, and which key it was does not matter — `ANYKEY` never asks. At a prompt it types its character and leaves it in the buffer, and `RETURN` is what hands the buffer over, which is how a yes-or-no question gets answered a letter at a time. |
| `WAIT <seconds>` | Say nothing for that long, which is how the adventure's own `TIME` is allowed to run out. |
| `DEBUG FLAG <n>` | Prints `FLAG_1=23`. |
| `DEBUG FLAGS` | All 256, one per line. `DEBUG FLAGS SET` for only the ones holding something. |
| `DEBUG AT` | `AT=1` — flag 38, where the player is. |
| `DEBUG PICTURE` | `PICTURE=1`, or `PICTURE=none` if the adventure has not asked for one. |
| `DEBUG OBJECTS` | `OBJ_12=254` for every object. |
| `DEBUG SAID` | What has been said and not matched. For writing the test, not for the test. |
| `ASSERT FLAG <n> = <v>` | Also `!=`, `>`, `<`, and the operator may be written up against its numbers. |
| `ASSERT AT = <n>` | Where the player is. |
| `ASSERT PICTURE = <n>` | The last illustration asked for. |
| `ASSERT SCORE = <n>` | Flag 30. |
| `ASSERT TURNS = <n>` | Flags 31 and 32, two bytes little-endian. |
| `ASSERT CARRIED\|WORN\|HERE\|PRESENT <object>` | Where an object is. `PRESENT` is carried, worn or in the room, which is what the condact of that name means. |
| `ASSERT OBJECT <n> AT <location>` | For a location that is not one of those four. |
| `ASSERT ENDED` | The adventure has to have ended by here. |
| `TRACE ON` / `TRACE OFF` | Turns the condact trace on and off, so `--trace` covers the stretch that matters instead of the whole run. |
| `SEED <n>` | Fixes the dice. |
| `TEST <name>` | Names the run, once, at the top. |
| `STEP <name>` … `END` | One step of the script. |
| `INCLUDE <file>` | Another script, beside this one. |
| `GAME <file>` | What to play, if the script wants to be runnable on its own. `-i` wins. |

`SEED`, `TEST`, `GAME` and `INCLUDE` are not steps: they are read when the file
is, and `SEED` is applied before the adventure starts, because by the time the
first step runs whatever `CHANCE` is in the opening has already been rolled.

## The flags of `qrun` that go with it

| Flag | What for |
|---|---|
| `-T`, `--test` | The script to play. Repeat it to play several. |
| `--show` | Print what the adventure says as well as what the steps did. Off by default, because a test is about the steps. |
| `--transcript FILE` | Everything the adventure said, kept whatever happens. The first thing to read when a step fails for a reason the report cannot show. |
| `--seed N` | Fix the dice for the whole run. Wins over a script's own `SEED`. |
| `--trace FILE` | The condact trace, as ever. With `TRACE` orders in the script it starts off and the script decides. |

## What it cannot do

No pictures and no character set: `qrun` is an interpreter, not a Spectrum. It
remembers which illustration was asked for — that is what `ASSERT PICTURE` reads
— and draws nothing. `SAVE`, `LOAD`, `RAMSAVE` and `RAMLOAD` do nothing, so a
script cannot checkpoint and come back; a branch is a second script with the
same `INCLUDE` at the top of it.
