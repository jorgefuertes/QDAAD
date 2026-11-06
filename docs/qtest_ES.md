# El lenguaje `.qtest`

*[In English](qtest.md)*

Un walkthrough no es código. Es una lista de qué teclear y qué debería contestar, escrita mientras se juega, y su sitio es un fichero que alguien pueda editar con el juego en la otra ventana. `qrun --test` lee ese fichero.

```sh
qdaad compile -i examples/AO/ZXSpectrum/+3/part1/game.sce -m pc -s vga -f ddb -o tmp/AO1
qrun -i tmp/AO1.ddb -T examples/AO/test/part1.qtest
```

Sale con cero si todos los pasos se cumplieron, y distinto de cero con la línea que no:

```text
╭─────────────────────────────────────╮
│ La Aventura Original, parte I       │
╰─────────────────────────────────────╯
▶ la portada y el bosque                    OK
▶ la botella, valle abajo y volver          OK
▶ las botas, al este y al sur              FAIL
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

El bloque que importa es el segundo: qué contestó la aventura a la última orden que el guion le dio, con esa orden nombrada encima. De ahí sale la frase que va en un `EXPECT` cuando se está escribiendo el guion — todo lo no emparejado son las sobras de cuantas órdenes hayan pasado, que es el bloque de debajo y el que hay que leer cuando un paso falla porque sus `EXPECT` están en orden equivocado. Las tiradas de líneas en blanco se aprietan en los dos, porque una aventura rellena con saltos donde a su autor le apetece.

Un caret anuncia cada paso al abrirlo, así que una ejecución que se cuelga ya ha dicho en qué paso estaba, y el veredicto cae al final de esa línea al llegar su `END` — verde para `OK`, rojo para `FAIL`, los dos del mismo ancho para que se lean como una columna. Lo que se diga por dentro del paso, un `DEBUG` o las palabras de la aventura con `--show`, rompe esa línea, y entonces la insignia baja al margen.

## El reloj, que es lo único de aquí que no es obvio

Una aventura que ha puesto `TIME` le da al jugador unos segundos para contestar, y al agotarse las tablas corren sin orden — «El tiempo pasa... ¡Y no vuelve!», o el enano que entra en la habitación. Eso es la mitad de lo que hace una aventura y tiene que poder probarse, así que **los segundos son segundos**: un paso que espera, espera, igual que un jugador que se queda quieto.

Decide quien se agote primero.

```qtest
EXPECT "El enano entra"       ; cinco segundos, que es lo que toca cuando
                              ; nadie dice otra cosa. La aventura concede 47,
                              ; así que se rinde él: el paso falla y la
                              ; partida no se ha movido un dedo

EXPECT "El enano entra", 60   ; ahora expira el plazo de la aventura mientras
                              ; el paso espera. El runtime corre su turno por
                              ; sus propios motivos, y lo que imprima puede
                              ; ser justo lo que se esperaba
```

Lo que no se hace nunca es informar de un plazo vencido que no ha vencido. No costaría nada y sería una mentira con consecuencias: la aventura correría un turno que no habría corrido y diría algo que un jugador no habría visto nunca, que es observar una partida alterándola.

Así que un `EXPECT` que falla paga sus segundos, y uno que acierta no cuesta nada — el texto que busca ya está en el buffer, porque la máquina dijo todo lo que tenía que decir antes de pedir una orden. Una tanda en verde va tan rápida como el intérprete.

Cinco segundos es lo que espera un `EXPECT` que no lo dice. `TIMEOUT` cambia eso para un guion entero, una sola vez y en cualquier parte del fichero:

```qtest
TIMEOUT 1     ; impaciente, para un guion que se está escribiendo y se corre mil veces
TIMEOUT 60    ; para una aventura que se toma su tiempo
```

Cuando la aventura no tiene reloj puesto sobre la pregunta —y eso incluye las preguntas de sí o no de `QUIT` y de `END`— no hay nada que esperar: está aguantando a que le den una orden y no dirá nada más hasta tenerla. Un `EXPECT` falla entonces al instante y lo dice, en vez de gastar segundos de verdad para no aprender nada.

## Cómo se turnan un guion y una máquina

La máquina manda y el guion es lo que ella bombea. Cada paso corre dentro de una de las tres llamadas que la máquina hace a una consola —una línea, una línea con reloj encima, una pausa— más una última pasada cuando la partida ha terminado. No hay hilos, así que una ejecución es repetible.

**Una pausa espera**, porque la aventura lo pidió. Un guion que no tenga nada que pulsar falla ahí, con la línea a la que llegó, en vez de que se le deje pasar: dejarlo pasar sería el arnés decidiendo que esa pausa no cuenta, y una pausa es conducta propia de la aventura — cuántas hay y dónde caen es parte de lo que un test de ella comprueba. Un walkthrough que deja de funcionar el día que alguien añade una es un walkthrough que se ha dado cuenta.

Por eso un guion se lee como una sesión. La Aventura Original pide tres teclas antes de empezar la partida, y contestarlas es así:

```qtest
STEP "la portada, y después el bosque"
  EXPECT  "?"                                ; pregunta antes de mostrar nada
  KEY     SPACE
  EXPECT  "AVENTURA ORIGINAL"                ; los créditos
  KEY     SPACE
  EXPECT  "LA RESPUESTA DEPENDE DE TU VALOR" ; de qué va el juego
  KEY     SPACE
  EXPECT  "Estás en el bosque"
END
```

## Pararse a jugar

`BREAKPOINT` le entrega la aventura a quien esté en el terminal, en medio de un guion, con la partida exactamente en el estado al que el guion la llevó. Para eso es: lo difícil de escribir un walkthrough es saber qué dice la aventura en el turno cuarenta, y la manera más barata de averiguarlo es estar allí.

```qtest
STEP "la llave, sacudida del árbol"
  INPUT  SACUDIR RAMAS
  EXPECT "ha caido al suelo"
END

BREAKPOINT        ; echar un ojo antes de escribir el trozo siguiente

STEP "y de ahí en adelante"
  INPUT  COGER LLAVE
  EXPECT "Has cogido la llave"
END
```

Lo que se entrega es la consola en la que juega una persona, no una imitación, así que el reloj de la aventura y sus pausas se comportan igual que sin guion de por medio. El guion no avanza una sola orden por muchos turnos que esa persona se tome, y lo que la aventura diga va a su pantalla y no al informe.

`CONTINUE TEST` lo termina. El arnés lee esa línea y no la entrega jamás —es una palabra de este lenguaje y no una orden del juego— y el guion sigue por la orden siguiente al `BREAKPOINT`. Teclearlo en una pausa no hace nada, porque una pausa quiere una pulsación y de una pulsación no se saca una línea; la salida está en el prompt siguiente.

Dos cosas que conviene saber antes de apoyarse en él. Lo que la persona dijo y lo que le contestaron se queda en el buffer que leen los `EXPECT`, porque la regla aquí es siempre que el buffer es lo dicho y no emparejado, y un breakpoint no es motivo para inventar una excepción. Y la partida se queda donde ella la dejó, así que a los pasos de después de un breakpoint se les pregunta por un estado al que el guion no llevó la aventura — que está bien mientras se escribe, y no es para dejarlo commiteado.

Sin nadie en un terminal no hay a quién entregarla, así que lo dice y sigue. Un guion con un `BREAKPOINT` dentro corre igual en una tanda.

## Nada de aquí cambia lo que hace la aventura

A la máquina no se le dice que está bajo un test y no tiene manera de averiguarlo. Lo que un guion le entrega es una línea, una pulsación o un jugador que no dijo nada — que es todo lo que le entrega una persona.

Ésa es la raya por la que está trazado el lenguaje, y vale la pena ser exacto sobre dónde cae el reloj en ella. Cuando un `EXPECT` con segundos deja que el plazo venza, **es el runtime quien corre el turno**: él puso el `TIME`, expiró su propio plazo, y la consola no hizo más que informar con verdad de que nadie contestó dentro del límite que la aventura misma eligió. Una persona quieta cuarenta y siete segundos produce lo mismo. Lo que el arnés no hace nunca es dar una orden que nadie escribió.

## Escribir uno

Una orden por línea, palabras clave en la caja que se quiera, `;` empieza comentario —igual que en un DSF—. Un argumento de texto es el resto de la línea, o lo que va entre comillas si empieza por una; las comillas son para cuando el texto lleva una coma, o espacios de los bordes que importan. Los ficheros van en UTF-8. Un texto entrecomillado lee `\n`, `\t`, `\"` y `\\`, y deja intacta cualquier otra barra invertida — un salto de línea vale la pena, porque un `TEST` o un `STEP` cuyo nombre son dos líneas hace mejor encabezado que uno largo.

Una orden es también un comando: `INPUT coger tortilla` teclea las palabras **y** pulsa return. `KEY` es la que teclea sin enviar, y el sitio que la quiere es una pregunta que se contesta con una sola letra — `KEY S` y luego `KEY RETURN`. Un `KEY RETURN` que nadie estaba esperando no se refusa: entrega una línea vacía, la aventura contesta con su «no entiendo», y esa contestación en la transcripción es cómo se nota.

```qtest
; La Aventura Original, segunda parte
TEST    "la segunda parte, entrada"
GAME    "../../../tmp/AO2.ddb"   ; opcional; -i manda sobre esto
SEED    1989
TIMEOUT 2                        ; lo que espera un EXPECT que no lo diga

INCLUDE "prologo.qtest"          ; la portada y la clave de acceso

STEP    "la primera habitación"
  EXPECT  "Los rayos del sol", 2
  ASSERT  AT = 1
  ASSERT  PICTURE = 1
END

STEP    "lo que la primera parte dejó en la mano"
  INPUT   i
  EXPECT  "Una llave"
  EXPECT  NOT "No tienes nada"
  ASSERT  CARRIED 12
END

STEP    "el tiempo pasa, y no vuelve"
  WAIT    50                     ; cincuenta segundos callado, de verdad
  EXPECT  "El tiempo pasa"
  DEBUG   FLAGS SET
END

STEP    "FIN lo termina"
  INPUT   fin
  EXPECT  "seguro"
  INPUT   s
  EXPECT  "otra partidita"
  INPUT   n
  ASSERT  ENDED
END
```

La sangría de dentro de un `STEP` es asunto de quien escribe; el lenguaje no la mira.

### Los dibujitos del nombre de un objeto

Una aventura de 1989 dibujaba una figurita al lado del nombre de un objeto, con la mitad gráfica de su juego de caracteres: «Una llave» y luego dos celdas que forman el dibujo de una llave. Aquí no hay juego de caracteres ni con qué dibujar, así que cada una de esas celdas sale nombrada por la ranura de la que se habría dibujado:

```text
> i
Tienes Una linterna encendida [UDG:48][UDG:49], Una moneda [UDG:50][UDG:51],
Una llave [UDG:44][UDG:45].
```

Para un guion eso significa una cosa: **espera las palabras, no el dibujo.** `EXPECT "Una llave"` acierta y seguirá acertando el día que alguien mueva el sprite a otra ranura; `EXPECT "Una llave [UDG:44][UDG:45]"` es para cuando el dibujo es lo que se está probando, que es pocas veces.

Los títulos son el mismo mecanismo y salen tal cual, porque allí los glifos son las letras dibujadas más grandes: `#gTIMACUS#t` es la clave de acceso y se lee como tal. Lo que distingue a los dos es la caja — las mayúsculas son una tipografía de titular, y las ranuras de minúscula, dígito y signo son donde el autor puso los sprites. De los 84 tramos gráficos de las cinco aventuras de Aventuras AD, 56 son palabras en mayúsculas y los otros 28 son uno o dos de ésos.

### El paso es la unidad del informe

`STEP` abre uno y `END` lo cierra, y cada `STEP` tiene que tener su `END`. Eso se comprueba al leer el fichero y no al correrlo: un `STEP` sin cerrar o un `END` suelto se dicen con su línea antes de arrancar siquiera la aventura. No se anidan, y tienen que cuadrar dentro de un mismo fichero, así que un `INCLUDE` trae pasos enteros y nunca la mitad de uno.

Quien imprime es `END`. Llegar hasta él significa que todo lo de dentro del paso se cumplió, así que ahí va el `OK`. Un fallo para la tanda — un walkthrough que va una respuesta por detrás contesta cada paso que queda en la habitación equivocada, y reporta disparates en diez sitios en vez de la verdad en uno.

## Las órdenes

| Orden | Qué hace |
|---|---|
| `EXPECT <texto>[, <segundos>]` | Espera el texto. Se busca como subcadena, sin distinguir mayúsculas, en lo dicho y no emparejado todavía; al acertar se come el buffer **hasta el final de la coincidencia**, que es lo que mete en el test el orden en que se dicen las cosas. Una tirada de espacios del texto casa con cualquier tirada de espacios de lo dicho, porque los saltos de línea de la máquina están donde los puso la base de datos. Los segundos son cuánto está dispuesto a esperar, de verdad, y son **5** si no se dicen — ver el reloj, arriba. |
| `EXPECT NOT <texto>` | Falla si el texto está en lo no emparejado. No se come nada, y no lleva segundos: va de lo que ya se ha dicho. |
| `INPUT <texto>` | Un comando: teclea el texto **y** pulsa return. Falla si no hay dónde teclear, o sea si la partida ha terminado. |
| `KEY <tecla>` | `RETURN`/`ENTER`, `SPACE`, `ESC`, o un carácter. En una pausa es la pulsación que la aventura espera, y cuál fuera da igual — `ANYKEY` no pregunta nunca. En un prompt teclea su carácter y lo deja en el buffer, y `RETURN` es quien entrega el buffer, que es cómo se contesta letra a letra a una pregunta de sí o no. |
| `WAIT <segundos>` | Callar ese tiempo, que es cómo se deja vencer el `TIME` de la aventura. |
| `DEBUG FLAG <n>` | Imprime `FLAG_1=23`. |
| `DEBUG FLAGS` | Los 256, uno por línea. `DEBUG FLAGS SET` para sólo los que valen algo. |
| `DEBUG AT` | `AT=1` — la bandera 38, dónde está el jugador. |
| `DEBUG PICTURE` | `PICTURE=1`, o `PICTURE=none` si la aventura no ha pedido ninguna. |
| `DEBUG OBJECTS` | `OBJ_12=254` para cada objeto. |
| `DEBUG SAID` | Lo dicho y no emparejado. Para escribir el test, no para el test. |
| `ASSERT FLAG <n> = <v>` | También `!=`, `>`, `<`, y el operador puede ir pegado a sus números. |
| `ASSERT AT = <n>` | Dónde está el jugador. |
| `ASSERT PICTURE = <n>` | La última ilustración pedida. |
| `ASSERT SCORE = <n>` | La bandera 30. |
| `ASSERT TURNS = <n>` | Las banderas 31 y 32, dos bytes en little-endian. |
| `ASSERT CARRIED\|WORN\|HERE\|PRESENT <objeto>` | Dónde está un objeto. `PRESENT` es llevado, puesto o en la habitación, que es lo que significa el condacto de ese nombre. |
| `ASSERT OBJECT <n> AT <localidad>` | Para una localidad que no sea una de esas cuatro. |
| `ASSERT ENDED` | La aventura tiene que haber terminado llegados aquí. |
| `BREAKPOINT` | Para y entrega la aventura a quien esté en el terminal, que juega desde ahí. `CONTINUE TEST` lo lee el arnés, no se le da nunca a la aventura, y devuelve el mando al guion en la orden siguiente al breakpoint. Sin nadie en un terminal lo dice y sigue, así que un guion con uno dentro corre igual en una tanda. |
| `TRACE ON` / `TRACE OFF` | Enciende y apaga la traza de condactos, para que `--trace` cubra el tramo que interesa y no la ejecución entera. |
| `TIMEOUT <segundos>` | Cuánto espera un `EXPECT` que no lo diga. Una vez por guion, y vale para todo él — también para los `EXPECT` que estén por encima de la línea. **5** cuando nadie lo dice. |
| `SEED <n>` | Fija los dados. |
| `TEST <nombre>` | Nombra la tanda, una vez, arriba. |
| `STEP <nombre>` … `END` | Un paso del guion. |
| `INCLUDE <fichero>` | Otro guion, al lado de éste. |
| `GAME <fichero>` | Qué jugar, si el guion quiere poder correrse solo. `-i` manda. |

`SEED`, `TIMEOUT`, `TEST`, `GAME` e `INCLUDE` no son pasos: se leen cuando se lee el fichero, y el `SEED` se aplica antes de que la aventura arranque, porque para cuando corre el primer paso ya se ha tirado el `CHANCE` que hubiera en la apertura.

## Las opciones de `qrun` que van con esto

| Opción | Para qué |
|---|---|
| `-T`, `--test` | El guion a jugar. Repetirla para jugar varios. |
| `--show` | Imprime lo que la aventura dice además de lo que hicieron los pasos. Apagado por omisión, porque un test va de los pasos. |
| `--transcript FICHERO` | Todo lo que la aventura dijo, guardado pase lo que pase. Lo primero que hay que leer cuando un paso falla por algo que el informe no puede mostrar. |
| `--seed N` | Fija los dados de toda la tanda. Manda sobre el `SEED` del guion. |
| `--trace FICHERO` | La traza de condactos, como siempre. Con órdenes `TRACE` en el guion arranca apagada y decide el guion. |

## Lo que no puede hacer

No hay imágenes ni juego de caracteres: `qrun` es un intérprete, no un Spectrum. Recuerda qué ilustración se pidió —eso es lo que lee `ASSERT PICTURE`— y no dibuja nada. `SAVE`, `LOAD`, `RAMSAVE` y `RAMLOAD` no hacen nada, así que un guion no puede guardar la partida y volver; una bifurcación es un segundo guion con el mismo `INCLUDE` en la cabecera.
