; A small adventure that does nothing but print xmessages.
;
; It is here because nothing else in the checkout uses them. The 66 editions
; that compare-drf and compare-drb chew on are all adventures of 1989 and
; XMES is a Maluva condact of 2020, so without a source written on purpose the
; two back ends would agree about a feature neither of them had been asked to
; build. Everything in here is chosen to make them disagree if they can:
;
;   - Enough texts to need more than one of the +3's 16 KiB slots, which is
;     what exercises the padding and the offsets past the first slot.
;   - Repeated words and endings, so that the texts take part in the token
;     table and the compressed bytes are what gets compared rather than the
;     spelling.
;   - XMESSAGE beside XMES, which differ by the line feed the compiler adds and
;     nothing else.
;   - A text with an accent, which is the one place a wrong character set shows
;     up as a different byte rather than a different letter.
;
; It is not meant to be played. Process 0 prints them and stops.

/CTL
_

/VOC
N        1   noun
S        2   noun
HABLAR   20  verb
MIRAR    21  verb
LEER     22  verb
CARTA    30  noun
LIBRO    31  noun
VIEJA    2   adjective

/STX
/0 "Está oscuro."
/1 "No puedo hacer eso."
/2 "No entiendo."
/3 "¿Y bien?"
/4 "No llevas nada."
/5 "Llevas:"
/6 "No hay nada aquí."
/7 "Aquí hay:"

/MTX
/0 "No puedes hacer eso."

/OTX
/0 "una carta vieja"

/LTX
/0 "La biblioteca. Hay un libro en la mesa, y una carta junto a él."
/1 "El pasillo. La biblioteca queda al norte."

/CON
/0
S 1
/1
N 0

/OBJ
/0  0 3 Y N  _ _ _ _ _ _ _ _ _ _ _ _ _ _ Y _  CARTA VIEJA

; The turn, written out because a version 3 interpreter has no loop of its own:
; it stacks process 0 and runs what it finds. This is the smallest one that
; plays — describe, read an order, answer it — so that the same source serves
; compare-drb and a .qtest. See examples/AO/.../v3-loop.sce for a real one.
/PRO 0
> _      _      DESC    @38

; PARSE reads the other way round from how it looks: it succeeds when the
; sentence made no sense, so this entry is the one that complains and the next
; is the one that answers. See the condact in internal/vm/condacts.go.
> _      _      PARSE   0
                SYSMESS 2
                REDO

> _      _      PROCESS 1
                REDO

; The answers, and every one of them an xmessage: that is what this source is
; for. The first XMES of each pair prints without a line break and the
; XMESSAGE after it with one, which is the whole difference between the two.
/PRO 1
> LEER   CARTA  XMESSAGE "La carta está escrita con una letra menuda y apretada."
                XMESSAGE "Dice que el libro de la mesa no es el que buscas."
                XMESSAGE "Y que el que buscas no está en la biblioteca."
                DONE

> MIRAR  LIBRO  XMESSAGE "El libro de la mesa no es el que buscas."
                XMES    "Las páginas están "
                XMESSAGE "en blanco."
                DONE

> HABLAR _      XMESSAGE "No hay nadie en la biblioteca a quien hablar."
                DONE

> _      _      SYSMESS 1
                DONE
/END
