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
;   mensaje 4  ->  11
;

/PRO 0

> _       _     AT      0
                PROCESS 6               ; inicialización

> _       _     WINDOW  1
                WINAT   0 0
                WINSIZE 24 42   ; la pantalla entera de un ZX

; CONJETURA: que el flag 0 marca la oscuridad y el objeto 0 es la lámpara. Es
; lo que hace la plantilla oficial; esta aventura puede llevarlo en otro sitio.
> _       _     CLEAR   28
                NOTZERO 0
                ABSENT  0
                SET     28

> _       _     NOTZERO 28
                SYSMESS 0

> _       _     ZERO    28
                DESC    @38

> _       _     PROCESS 3

> _       _     PROCESS 1

/PRO 1

> _       _     PROCESS 4

; La ventana vuelve a ser la nuestra antes de preguntar, porque las tablas de
; la aventura cambian de ventana y no todas la devuelven. La Aventura Original
; deja puesta la 5 —"WINDOW 5 / WINAT 0 5 / WINSIZE 25 36"— y el prompt salía
; en ella: en la columna 5 y en la fila 0, encima de la primera línea de la
; descripción, que se perdía entera.
;
; Sin WINAT ni WINSIZE a propósito. Cada ventana lleva su propio cursor, así
; que seleccionarla basta para seguir donde la descripción lo dejó; fijarla
; otra vez lo devolvería a 0,0, que es exactamente el defecto que esto arregla.
> _       _     WINDOW  1

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

; La ventana gráfica se queda sin superficie, porque aquí no hay ilustraciones:
; la versión 3 no dibuja vectoriales, que es lo único que trae una aventura de
; Spectrum de 1989. Sin esto, la ventana 0 conserva el tamaño que le da el
; intérprete y encima de la nuestra, así que un "WINDOW 0 / CLS" de la aventura
; —La Aventura Original lo hace al entrar en la depresión de 7 metros— borra la
; primera línea de la descripción que acabamos de escribir.
;
; La plantilla oficial no lo padece porque pone su texto en "WINAT 14 0", debajo
; del hueco de la imagen. Nosotros no podemos permitírnoslo: serían 14 filas de
; 24 perdidas para una imagen que no va a existir.
> _       _     WINDOW  0
                WINAT   0 0
                WINSIZE 0 0

> _       _     WINDOW  1
                WINAT   0 0
                WINSIZE 24 42   ; la pantalla entera de un ZX
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
;
; Sin el "LET 1 0" que la plantilla pone aquí. El flag 1 es cuántos objetos
; lleva el jugador y lo mantiene el intérprete, así que ponerlo a cero sólo
; acierta si la aventura empieza con las manos vacías. La plantilla sí, pero
; la segunda parte de La Aventura Original te entrega la lámpara, una llave y
; la comida con que acabó la primera, y entonces el primer INVENTARIO de la
; partida contesta "no llevas nada" con cuatro cosas en la mano. El RESET de
; la línea de arriba ya deja la cuenta como toca.
> _       _     RESET
                LET     52 10
                LET     37 4
                SET     46
                SET     47
