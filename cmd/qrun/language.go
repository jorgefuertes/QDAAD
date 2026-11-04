package main

// What --test reads, written out in the help.
//
// It is here rather than only in docs/qtest.md because the person who needs it
// is at a terminal with a script half written, and sending them to a file in
// the repository for the name of an order is sending them away. The reference
// is short enough to read on a screen, and anything longer than that belongs
// in the document.
const testLanguage = `An interpreter rather than an emulator: there is no Spectrum behind it, only
the adventure. It plays a version 3 database on the terminal, which is how a
change to the compiler gets tried without an emulator in the way. There are no
pictures and no character set, so what it is good for is the parser, the flags
and the process tables — and TIME, because a question with a clock on it is a
thing adventures do and a tedious thing to try by hand.

   qrun -i GAME.ddb

TESTING A WHOLE ADVENTURE

A .qtest script is a walkthrough: what to type and what should come back.
One order per line, keywords in any case, ';' starts a comment. A text
argument is the rest of the line, or what is between quotes if it starts with
one — quote it when it has a comma or edge spaces that matter. UTF-8.

   TEST   "La Aventura Original, the forest"
   SEED   1989

   STEP   "the title screen, and then the forest"
     EXPECT  "?"                     ; it asks for a key before showing anything
     KEY     SPACE
     EXPECT  "AVENTURA ORIGINAL"     ; the credits, and it asks again
     KEY     SPACE
     EXPECT  "LA RESPUESTA DEPENDE DE TU VALOR"
     KEY     SPACE
     EXPECT  "Estás en el bosque"
     ASSERT  AT = 3
   END

   STEP   "the bottle, four rooms down the valley"
     INPUT   S
     INPUT   COGER BOTELLA
     EXPECT  "Has cogido la botella"
     EXPECT  NOT "Aquí no hay de eso"
     ASSERT  CARRIED 4
   END

   STEP   "the time passes, and does not come back"
     WAIT    50              ; fifty seconds of saying nothing, for real
     EXPECT  "El tiempo pasa"
   END

STEP opens a step and END closes it. A caret announces the step as it opens
and the verdict lands at the end of that line when the END is reached, which
is the proof that everything in it held:

   ▶ the title screen, and then the forest     OK
   ▶ the boots, east and south                FAIL
     part1.qtest:63:3 EXPECT "Has cogido la pareja de botas"
     it did not say that in 5 seconds

A step that does not hold names the line it is written on, shows what the
adventure really said instead, and stops the run. Every STEP needs its END,
they do not nest, and they balance inside one file, which is why an INCLUDE
brings whole steps.

THE ORDERS

   EXPECT <text>[, <seconds>]  Wait for the text. Case is ignored, and a run
                               of space matches any run of space. A hit eats
                               what was said up to the end of the match, so
                               the order things are said in is tested too.
                               The seconds are how long it will wait, really;
                               5 if omitted.
   EXPECT NOT <text>           Fails if it has been said. Eats nothing.
   INPUT <text>                One command: types the text and presses
                               return.
   KEY <key>                   RETURN, ENTER, SPACE, ESC or one character. At
                               a pause it is the keypress; at a prompt it
                               types and leaves it in the buffer, and RETURN
                               hands the buffer over. KEY S then KEY RETURN
                               is how a yes-or-no question is answered.
   WAIT <seconds>              Say nothing for that long, which is how the
                               adventure's own TIME is let run out.
   DEBUG FLAG <n>              Prints FLAG_1=23.
   DEBUG FLAGS [SET]           All 256, one per line; SET for the ones
                               holding something.
   DEBUG AT                    AT=1, which is flag 38.
   DEBUG PICTURE               PICTURE=1, or PICTURE=none.
   DEBUG OBJECTS               OBJ_12=254 for every object.
   DEBUG SAID                  What has been said and not matched yet.
   ASSERT FLAG <n> = <v>       Also !=, > and <.
   ASSERT AT|PICTURE|SCORE|TURNS = <v>
   ASSERT CARRIED|WORN|HERE|PRESENT <object>
   ASSERT OBJECT <n> AT <location>
   ASSERT ENDED                The adventure has to have ended by here.
   TRACE ON | TRACE OFF        Bounds --trace to the stretch that matters.
   SEED <n>                    Fixes the dice.
   TEST <name>                 Names the run, once, at the top.
   STEP <name> ... END         One step of the script.
   INCLUDE <file>              Another script, beside this one.
   GAME <file>                 What to play, if the script is to be runnable
                               on its own. --input wins over it.

THE CLOCK

The seconds are seconds: a step that waits waits, the same as a player sitting
still. Whoever runs out first decides. An EXPECT given five seconds against an
adventure that allows forty-seven gives up first, so the step fails and the
game has not moved; given sixty, the adventure's own timeout fires while the
step is still waiting, the runtime runs its turn for its own reasons, and what
it prints may be the very thing being waited for — which is how "El tiempo
pasa... ¡Y no vuelve!" and the dwarf walking into the room become testable.

A failing EXPECT pays its seconds. A passing one costs nothing: what it is
looking for was said before the adventure asked for an order. And with no clock
running on the question there is nothing to wait for at all, so it fails at
once and says so.

NOTHING HERE CHANGES WHAT THE ADVENTURE DOES

The machine is not told that it is being tested and has no way of finding out.
What a script hands it is a line, a keypress, or a player who said nothing,
which is all a person hands it too. Where the clock runs out it is the runtime
running the turn — it set the TIME and its own timeout expired — and the
console only reported that nobody answered in time.

So a pause waits, because the adventure asked it to. A script with nothing to
press fails there rather than being let through: how many pauses there are and
where they fall is the adventure's own behaviour, and part of what a test of it
is for. That is what the three keys at the top of the example are.

   qrun -i GAME.ddb -T part1.qtest
   qrun -i GAME.ddb -T part1.qtest --show --transcript said.txt

See docs/qtest.md for the whole of it, and examples/AO/test for La Aventura
Original played to the end.`
