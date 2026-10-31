; El bucle del juego, que una base de datos de formato 1 no trae.
;
; En DAAD 1 el bucle estaba en el código del intérprete: describir, pedir la
; orden, parsearla, ejecutar la tabla de respuestas, mover al jugador. El
; intérprete v3 no tiene bucle propio —apila el proceso 0 y ejecuta lo que
; encuentre—, así que hay que escribirlo, y esto es el de la plantilla oficial
; con los gráficos quitados.
;
; ATENCIÓN: esto no es una traducción de nada. Lo demás que emite la conversión
; dice en términos de v3 lo que el binario ya decía; esto dice algo que el
; binario nunca dijo. Las líneas marcadas con CONJETURA son valores que la
; plantilla sabe de su propio juego y que de esta aventura no se saben.
;
; Dónde ha ido cada tabla de la aventura:
;
;   tabla 0  ->  5   la tabla de respuestas
;   tabla 1  ->  3   tras describir la localidad
;   tabla 2  ->  4   cada turno
;   tabla 3 en adelante  ->  7 en adelante
;
; Y qué mensajes de sistema se han movido, porque el intérprete v3 los
; imprime por su cuenta como separadores:
;
;   mensaje 3  ->  34
;   mensaje 4  ->  22
;

/PRO 0

> _       _     AT      0
                PROCESS 6               ; inicialización

> _       _     WINDOW  1
                WINAT   0 0
                WINSIZE 24 42

; CONJETURA: que el flag 0 marca la oscuridad y el objeto 0 es la lámpara. Es
; lo que hace la plantilla oficial; esta aventura puede llevarlo en otro sitio.
> _       _     CLEAR   28
                NOTZERO 0
                ABSENT  0
                SET     28

> _       _     NOTZERO 28
                SYSMESS 0

; La pantalla se limpia cada turno, y con ella el dibujo del turno anterior.
; Sin esto la imagen de la localidad que acabas de dejar se queda puesta y el
; texto de la nueva se escribe encima.
;
; Y de paso el flag 40, que es como el intérprete del Spectrum de 1989 le decía
; a la aventura si esta localidad tenía dibujo. Lo ponía él y por eso ninguna
; de las dos aventuras lo escribe nunca: sólo lo leen. La Aventura Original
; pinta con "ZERO 40" su imagen genérica —el dibujo 27, el logotipo de AD entre
; los árboles— y El Jabato limpia con él la ventana gráfica. Sin reponerlo, el
; "ZERO 40" se cumple siempre y la genérica tapa la imagen de cada localidad.
> _       _     WINDOW  1
                WINAT   0 0
                WINSIZE 24 42
                CLS
                CLEAR   40

; Y la imagen de donde estés, si la hay. PICTURE no dibuja: busca la imagen y
; la deja preparada, y además es una condición —así que una localidad sin
; dibujo se lleva por delante el resto de la entrada, que es justo lo que se
; quiere—. Quien la pinta es DISPLAY 0. Ver _internal_picture y
; _internal_display, daad_condacts.c:2609 y 2631.
;
; La indirección es lo que hace que valga para las cuarenta y una: @38 es la
; bandera de la localidad, así que cada sitio pide la suya.
;
; El texto baja a la fila 14 porque es lo que miden 32 de las 41 imágenes de
; esta parte. En 1989 no era una fila fija: cada dibujo declara su alto en el
; byte 3 de su registro y el intérprete escribía debajo (09-graficos.md §9.6),
; así que los pocos más bajos dejan aquí un hueco y los 3 de 22 filas se
; solapan. Lo segundo no se nota: uno es el marco de la intro, que escribe en
; su propia ventana 5.
> _       _     NOTAT   0
                PICTURE @38
                DISPLAY 0
                SET     40
                WINAT   14 0
                WINSIZE 10 42

; Y si esta localidad no tiene dibujo propio, la ventana baja igual, porque la
; aventura va a pintar arriba su imagen genérica —el "ZERO 40" de la tabla 3— y
; si no, el texto le cae encima. Va en una entrada aparte y no en la de arriba:
; el DISPLAY tiene que correr con la ventana todavía a pantalla completa.
;
; El NOTAT 0 de las dos está por la pantalla de arranque. La localidad 0 no es
; un sitio: es donde la aventura pregunta antes de empezar, y lo hace con
; "PRINTAT 11 21" sobre la pantalla entera. Con la ventana ya encogida a las
; filas 14-23 esa fila 11 no existe y el "?" acababa donde podía.
> _       _     NOTAT   0
                ZERO    40
                WINAT   14 0
                WINSIZE 10 42

> _       _     ZERO    28
                DESC    @38

> _       _     PROCESS 3

> _       _     PROCESS 1

/PRO 1

> _       _     PROCESS 4

; Una frase sin verbo deja fVerb a un valor que las entradas comodín de una
; tabla de respuestas de DAAD 1 toman por una dirección, porque comprueban
; "LT 33 14". El 255 es lo que el intérprete de 1989 dejaba ahí, y hace falta a
; los dos lados del PARSE: antes porque si el parser no lo toca queda lo del
; turno anterior, y después porque cuando no hay verbo lo pone a cero.
> _       _     LET     33 255

> _       _     PARSE   0
                PROCESS 2
                REDO

> _       _     ZERO    33
                LET     33 255

> _       _     EQ      31 255
                PLUS    32 1
                CLEAR   31
                SKIP    1

> _       _     PLUS    31 1

> _       _     PROCESS 5
                ISDONE
                REDO

> _       _     MOVE    38
                CLS
                RESTART

> _       _     NEWTEXT
                LT      33 14
                SYSMESS 7
                REDO

> _       _     SYSMESS 8
                REDO

/PRO 2

> _       _     HASAT   49
                SYSMESS 35
                DONE

> _       _     SYSMESS 6

/PRO 6

> _       _     WINDOW  1
                WINAT   0 0
                WINSIZE 24 42
                CLS
                SET     255

$limpia
> _       _     MINUS   255 1
                NOTEQ   255 29
                CLEAR   @255

> _       _     NOTZERO 255
                SKIP    $limpia

; CONJETURA: la fuerza y cuánto se puede llevar. Son los valores por defecto de
; la plantilla, no los de esta aventura.
> _       _     RESET
                LET     52 10
                LET     37 4
                SET     46
                SET     47
                LET     1 0
