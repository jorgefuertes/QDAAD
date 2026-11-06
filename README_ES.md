# QDAAD

Reimplementación en Go del sistema DAAD de creación de aventuras conversacionales.

Cuatro herramientas, y entre ellas un viaje de ida y vuelta: **leer una aventura de la cinta en que salió en 1989 y escribir una cinta que se pueda jugar**.

| | |
|---|---|
| `qundaad` | lee una base de datos DAAD —o la imagen de disco o cinta que la contiene— y devuelve el fuente que la produjo |
| `qdaad` | compila ese fuente y vuelve a dar una base de datos |
| `qpack` | empaqueta una compilación en la cinta, el disco o los ficheros que carga cada máquina |
| `qrun` | ejecuta una base de datos aquí, sin emulador |

El viaje de ida y vuelta no es una afirmación, es un objetivo del Makefile. En `examples/AO/` está La Aventura Original descompilada, convertida de la versión 1 a la 3 y corregida a mano donde la conversión todavía no llega; `make example-ao` la construye en dos cintas de 48K y dos discos de +3. La parte 1 del disco se ha jugado entera en ZEsarUX, y la 2 hasta donde se ha llegado. Ver [su README](examples/AO/README.md), en inglés.

## Qué se comprueba, y contra qué

Tres de estas comparan nuestra salida con la de las herramientas de referencia, byte a byte. La referencia es DAAD Reborn, de Uto, que no ha tenido nada que ver con escribir esto.

| `make` | | |
|---|---|---|
| `decomp-all` | descompila todas las ediciones y las contrasta entre sí | **38 parejas** |
| `recompile-all` | le pasa cada descompilación al compilador de referencia | **66 aceptadas** |
| `compile-all` | convierte cada edición a la versión 3 y la vuelve a compilar | **66 compiladas, 0 fallos** |
| `compare-drf` | lee cada fuente con los dos analizadores y compara lo que entendió cada uno | **70 iguales, 0 distintos** |
| `compare-drb` | enlaza cada fuente con los dos generadores y compara las bases de datos | **198 iguales, 0 distintos** |
| `compare-daadmaker` | empaqueta cada fuente en una cinta de 48K con los dos empaquetadores | **63 iguales, 0 distintos** |

`compare-drb` cubre tres destinos por fuente, para que el orden de bytes y el relleno queden ejercitados. Lo que salta es lo que la referencia no enlaza; lo que salta `compare-daadmaker` son los fuentes en idiomas que su compilador no habla y los que no caben en un Spectrum de 48K.

## qundaad, el descompilador

```sh
go build -o dist/qundaad cmd/qundaad/qundaad.go

qundaad decompile --input PART1.DDB              --output src/
qundaad decompile --input ORIGINAL.ADF           --output src/
qundaad decompile --input "Aventura Original.dsk" --output src/

# solo lo legible: sin copias de los binarios originales
qundaad decompile --input ORIGINAL.ST --output src/ --no-binaries
```

La entrada puede ser la base de datos suelta o **la imagen del disco o la cinta original**. Una imagen se abre, se recorre y cada base de datos que haya dentro se descompila en su propio directorio.

La salida es UTF-8, legible y modificable hoy, repartida en un fichero por sección y unida con `#include` desde `game.sce` — que es como se organizaban los fuentes de la época: el manual de 1991 avisa de los peligros de «incluir un fichero TOK que contenga otro `/TOK`», advertencia que solo tiene sentido si trocear era lo normal.

**Es fuente que un compilador acepta**, no un listado, y no sólo el nuestro: `recompile-all` le pasa cada descompilación a DAAD Reborn Compiler 0.40 y las 66 se aceptan, devolviendo la codificación a ISO-8859-1 y dejando fuera la sección `/TOK`, que la versión 3 quitó.

Tres de ellas se aceptan y no hay que fiarse: las tablas de proceso de Cozumel en Spectrum siguen sin poder leerse, y su fuente lo dice en la cabecera. Compilarlas daría un juego, pero no ese.

Junto al fuente van dos directorios más: `chr/` con las tipografías y `gfx/` con las imágenes. De cada cosa se guarda **el binario tal como venía y su conversión a PNG**, y de los archivos de ilustraciones también cada dibujo por separado, con un índice que dice qué localidades usan cada uno y con qué paleta. Con `--no-binaries` se dejan solo las conversiones, que ahorra la mitad del espacio.

### Lo que deduce en vez de suponer

Una `.DDB` no declara casi nada de lo que hace falta para leerla. `qundaad` no se fía de tablas indexadas por plataforma —de ahí que las herramientas de referencia den una base de datos de PC por *little-endian* cuando no lo es— y lo saca de los propios datos:

- **Orden de bytes y tamaño de cabecera**, probando las cuatro combinaciones y quedándose con la que hace que la cabecera se describa a sí misma. En estas cinco aventuras aparecen las cuatro.
- **La dirección de carga**, cuando la base de datos no es un fichero sino que está incrustada en un programa o puesta a pelo sobre un disco. Sus punteros son direcciones absolutas, y el vocabulario delata dónde se cargó todo — pero solo va *casi* pegado a la cabecera, porque algunas compilaciones dejaron relleno entre las dos: 6 bytes en el Amstrad PCW y 26 en Los templos sagrados. Así que se prueba cada anchura de relleno que tenga una palabra al final, y decide la lectura que cuadra. De ahí sale el `0x100` del PCW, que es donde CP/M carga un programa.
- **Si hay tabla de compresión**, que es opcional: la versión CGA de El Jabato se compiló sin ninguna.

### Soportes que abre

`internal/media` lee el contenedor de cada máquina, porque una base de datos casi nunca es un tramo contiguo de bytes en el medio original:

| Soporte | |
|---|---|
| Amiga `.ADF` | OFS (con cabecera en cada bloque de datos) y FFS |
| Atari ST y MSX `.ST`, `.DSK` | FAT12 |
| Commodore `.d64` | CBM DOS, sectores encadenados |
| Amstrad y Spectrum `.DSK` | CPCEMU, variante original y extendida |
| Amstrad CPC, PCW y Spectrum +3 | CP/M, cuando el disco lleva directorio |
| Cintas `.TZX` y `.CDT` | los 25 tipos de bloque del formato |
| Commodore `.TAP` | no: guarda los pulsos, no los bytes |

Los discos `.DSK` vienen **de dos maneras, y el contenedor no dice cuál**. Unos son volúmenes CP/M corrientes: los del Amstrad PCW nombran sus ficheros, y resulta que el disco de Spectrum +3 de Cozumel también. Otros se formatearon para el cargador del propio juego y no son más que una tira de sectores. La prueba es leer el directorio — y tiene que ser una prueba severa, porque un disco de programa y de gráficos tiene sectores que pasan por directorio si se les deja.

Donde no lo hay, se buscan las bases de datos por firma entre los sectores y se las obliga a demostrar que lo son —cabecera coherente, un vocabulario cuyas palabras son palabras, tablas de texto y conexiones que leen, y prosa que parece prosa— antes de aceptarlas.

## Las imágenes y las tipografías

Las tipografías son 2176 bytes: **cabecera AMSDOS de 128 y 256 glifos de ocho filas**. Que lleven la cabecera de Amstrad hasta en la versión de PC dice de dónde salió la herramienta — Aventuras AD trabajaba en CPC y PCW. Las letras se dibujan en seis columnas de las ocho; los glifos del 128 en adelante no son letras sino **tramas de sombreado**, y esas sí usan las ocho.

De ahí sale además la confirmación del juego de caracteres de DAAD leída en los propios bitmaps, no en el código de otras herramientas: 16 = ª, 17 = ¡, 18 = ¿, 19 = «, 20 = », 21 a 25 = á é í ó ú, 26 = ñ, 27 = Ñ, 28 = ç, 29 = Ç, 30 = ü, 31 = Ü.

Las **portadas** se dibujan en las cuatro rutas: CGA modo 4 con sus bancos entrelazados, EGA modo 0Dh en cuatro planos, y Degas de 68000 — que en Atari entrelaza los planos por palabra y en Amiga los guarda enteros, cosa que el fichero no dice y hay que tomar del soporte.

Hay **tres compresiones**, una por generación, y ninguna es un formato conocido. La del PC salió del propio compresor: `DMG.EXE`, el gestor de gráficos que construyó estos archivos, sigue viniendo con la entrega de 1991, y su rutina en `0x3b4f` cuenta cuántas veces sale cada valor de byte en la imagen, se queda con los cuatro más frecuentes y escribe la imagen byte a byte — salvo que un byte de esos cuatro va seguido de la cuenta de cuántas veces se repite. Cuántos de los cuatro reciben ese trato se decide probando los cuatro y quedándose con el que menos ocupa. Al leerla, 189 de las 191 comprimidas dan exactamente el tamaño que dice su cabecera habiéndose comido el flujo entero.

Las otras dos son las de 68000, y ninguna cede a ningún esquema estándar: cada píxel son cuatro bits, y la cabecera lleva **una máscara de dieciséis bits que dice qué colores van seguidos de una repetición**. Como el significado de los cuatro bits siguientes depende del color recién leído, no hay esquema estándar que encaje. Salió leyendo los intérpretes con `objdump -m m68k`: la rutina `0x20aa` del Atari de La Aventura Original, y la `0x269a` de Los Templos Sagrados.

Hay dos generaciones de archivo y el fichero no dice cuál es. Las aventuras tardías ensancharon la ranura de 44 a 48 bytes y cambiaron de dónde se sacan los píxeles —un nibble de un longword en vez de un bit de cada uno de cuatro bytes—, y sus ediciones de PC llevan el mismo archivo con los bytes al revés. Se prueban las formas y se queda la que cuadra: un archivo pone su primera imagen justo donde acaba su tabla de ranuras, y solo la lectura correcta la deja ahí.

Ahora mismo se convierten **1798 imágenes**: tipografías, portadas e ilustraciones, en todas las máquinas cuyas bases de datos se pueden leer.

## qdaad y qpack, el camino de vuelta

```sh
go build -o dist/qdaad ./cmd/qdaad
go build -o dist/qpack ./cmd/qpack

qdaad compile -i game.sce -o build/ -f build -m zx -s 48k -v 3 --part 1
qpack -f tap -o GAME --name GAME build/      # GAME.tap
```

Las dos eran una sola orden y se separaron porque son dos trabajos. Compilar va de un fuente y da una base de datos; empaquetar va de una **release**, que va de una máquina. `qpack` escribe `tap`, `tzx`, `dsk`, `dir` —todo lo que la máquina necesita como ficheros sueltos— y `qrun`, nuestro propio contenedor.

Lo que entra en una release además de la base de datos lo declara el fuente, así que el mismo directorio de compilación sirve para cualquiera de ellos:

```text
; qdaad: charset=gfx/chr/udg.chr    la tipografía, que la versión 3 no trae
; qdaad: images=gfx/scr             un directorio de pantallas de Spectrum
; qdaad: loading=gfx/loading.scr    la imagen que se enseña mientras carga el resto
```

Una **cinta de 48K** es un solo flujo: un cargador en BASIC que aparta RAMTOP, y detrás la pantalla de carga, el intérprete, la base de datos, el bloque de gráficos y tantas imágenes como quepan en la memoria que queda. El cargador está escrito y no prestado, y son los mismos bytes que escribe daadmaker, hasta el espacio sobrante de una de sus líneas —que es lo que hace que `compare-daadmaker` signifique algo—. Un `tzx` son los mismos bloques envueltos, 46 bytes más gordo.

Un **disco de +3** son los ficheros sueltos, y va escrito en el formato propio del +3 y no en el DATA del CPC, que es el que sigue llevando un disco de CPC. La diferencia son dieciséis bytes en el primer sector de la pista 0 —la especificación del disco— y la razón es el byte que hace 7 de ella: cuántos bloques ocupa el directorio. Un disco DATA no lleva especificación, así que la respuesta son 64 nombres y no hay discusión; diciendo cuatro bloques en vez de dos son 128, por dos kilobytes de un disco al que le sobran ochenta. Una imagen es un fichero y un fichero es un nombre, así que en una aventura con un dibujo por localidad lo que se acaba antes son los nombres.

La especificación es la del disco de la propia Aventura Original, leída de él, con ese byte cambiado. Verificado en un +3: el menú carga el disco, un `CAT` saca todos los nombres, y el espacio libre que dice es la aritmética de los campos que se movieron y no la de sus valores por omisión.

## qrun, el intérprete

```sh
go build -o dist/qrun ./cmd/qrun
qrun -i GAME.qrun
```

Ejecuta una base de datos de versión 3 aquí, en el terminal, que es como se prueba un cambio del compilador sin un emulador de por medio. No es un Spectrum: no hay imágenes ni tipografía, así que para lo que vale es para el analizador, las banderas y las tablas de proceso. `TIME` y su plazo funcionan, porque una pregunta con un reloj encima es algo que las aventuras hacen y que a mano se prueba fatal.

### Probar una aventura entera: `.qtest`

Un walkthrough no es código. Es una lista de qué teclear y qué debería contestar, escrita mientras se juega, así que vive en un fichero y no en un test de Go:

```sh
qrun -i tmp/AO1.ddb -T examples/AO/test/part1.qtest
```

```qtest
TEST "el bosque, y la botella"
SEED 1989

STEP "la portada, y después el bosque"
  EXPECT  "?"                 ; pide una tecla antes de mostrar nada
  KEY     SPACE
  EXPECT  "AVENTURA ORIGINAL" ; los créditos, y vuelve a pedirla
  KEY     SPACE
  EXPECT  "LA RESPUESTA DEPENDE DE TU VALOR"
  KEY     SPACE
  EXPECT  "Estás en el bosque"
  ASSERT  AT = 3
END

STEP "la botella, cuatro salas valle abajo"
  INPUT   S
  INPUT   COGER BOTELLA
  EXPECT  "Has cogido la botella"
  EXPECT  NOT "Aquí no hay de eso"
  ASSERT  CARRIED 4
END

STEP "el tiempo pasa, y no vuelve"
  WAIT    50              ; cincuenta segundos callado, de verdad
  EXPECT  "El tiempo pasa"
END
```

Un caret anuncia cada paso al abrirlo y el veredicto cae al final de esa línea al llegar su `END`, que es la prueba de que todas las órdenes de dentro se cumplieron:

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

▶ 2 steps passed
▶ 1 step failed
```

Un paso que no se cumple para la tanda y `qrun` sale distinto de cero. Una orden es un comando —`INPUT` teclea sus palabras y pulsa return— y `KEY` es la que teclea sin enviar, para el único sitio que la quiere: una pregunta que se contesta con una letra. El lenguaje entero son `EXPECT`, `EXPECT NOT`, `KEY`, `INPUT`, `WAIT`, `DEBUG`, `ASSERT`, `BREAKPOINT`, `TRACE`, `TIMEOUT`, `SEED`, `INCLUDE` y `GAME` — `qrun --help` trae la referencia, y el documento del lenguaje todo lo demás: **[docs/qtest_ES.md](docs/qtest_ES.md)**, o [in English](docs/qtest.md).

`BREAKPOINT` es el que hace llevadero escribir un walkthrough: entrega la aventura a quien esté en el terminal, en medio de un guion, con la partida exactamente en el estado al que el guion la llevó. `CONTINUE TEST` la devuelve, y el guion sigue por la orden siguiente al breakpoint.

Nada de esto cambia lo que hace la aventura, y donde más se nota es en las pausas: la aventura pidió una tecla, así que el guion pulsa una. Decidir que esa pausa no cuenta sería el arnés cambiando el juego, y cuántas hay es parte de lo que un test de ella comprueba.

Los segundos son segundos, y un paso que espera, espera. Así es como «El tiempo pasa... ¡Y no vuelve!» y el enano que entra en la habitación se pueden probar: el guion se calla el tiempo suficiente para que venza el `TIME` de la aventura, y el runtime corre ese turno por sus propios motivos. Informar de un plazo vencido que no venció no costaría nada y haría que la aventura corriese un turno que no habría corrido — observar la partida alterándola. Un `EXPECT` que falla paga sus segundos; uno que acierta no cuesta nada, porque lo que busca ya se dijo antes de que la aventura pidiera una orden.

En `examples/AO/test` está la primera parte de La Aventura Original ganada —84 órdenes, una conversación, una cadena de cuatro fases y la clave de acceso que imprime al final— y la segunda recorrida. `make qtest` compila las bases de datos y las juega; la suite las juega también, contra el mismo walkthrough que usan los tests de Go, así que si alguna vez no coinciden, uno de los dos miente.

## Con qué se ha probado

Las cinco aventuras de Aventuras AD, en todas las ediciones que se han conseguido: **50 ficheros de soporte, de los que salen 66 bases de datos**.

| Aventura | PC | Amiga | Atari ST | Amstrad CPC | ZX Spectrum | PCW | MSX | C64 |
|----------|----|-------|----------|-------------|-------------|-----|-----|-----|
| La Aventura Original | sí | sí | sí | sí | sí | — | — | no |
| El Jabato | sí | sí | sí | no | sí | — | no | no |
| Cozumel | sí | sí | sí | sí | sí | sí | no | no |
| Chichén Itzá | sí | sí | sí | sí | sí | — | **sí** | no |
| Los templos sagrados | sí | sí | sí | sí | sí | sí | no | no |

Cada casilla dice si la aventura se lee en esa máquina, sus dos partes, no si se lee cada fichero que se distribuyó para ella. La raya significa que no se ha encontrado edición para esa máquina, y por eso la columna del PCW está casi vacía: solo dos de las cinco se vendieron para él.

Conviene detallar tres huecos. Los dos de El Jabato están en su fila: su segundo disco de Atari ST lleva un único fichero, el archivo de ilustraciones de la parte 2, así que ahí no hay base de datos que leer, y sus cintas de Spectrum no se leen, aunque el disco de esa misma máquina da las dos partes. El tercero es la columna entera del **Commodore 64**, y es un solo problema y no cinco: cada `.d64` lleva un único programa autoarrancable con su carga empaquetada, a entre 6,96 y 7,86 bits por byte frente a los 6,71 de una base de datos en claro. Ahí no se encontrará nada hasta identificar y deshacer el empaquetador.

La columna del **MSX** tiene un sí y ha sido el más costoso. Esos discos son de MS-DOS corrientes, pero ningún fichero suyo lleva una base de datos: cada pieza carga en el mismo búfer y se copia a sí misma abajo, a la RAM donde el juego corre de verdad, así que lo que se lee es la memoria que esas copias construyen. Está en `msx.go`. Cozumel y El Jabato todavía no salen —Cozumel reparte su memoria de otra manera y el disco de Jabato no da ni un fichero— y las cintas `.CAS` de las cinco necesitan un lector que no existe.

### Cómo se comprueba que el resultado es correcto

`make decomp-all` descompila las cinco aventuras y contrasta las ediciones entre sí: **38 emparejamientos**. Quince tienen que dar el mismo fuente entero, diecinueve tienen que coincidir en el texto y los datos, dos tienen que trazar los mismos dibujos, y dos tienen que describir el mismo mundo con otras palabras —la edición de MSX de Chichén Itzá contra la de PC, cuya prosa se reescribió para una pantalla de cuarenta columnas pero cuyos objetos, textos de objeto y conexiones no cambiaron—.

Es una comprobación fuerte porque los binarios comparados no se parecen en nada —distinto orden de bytes, distinto tamaño, distintos desplazamientos— y aun así tienen que converger en el mismo texto. Si se rompe la deducción del orden de bytes, la tabla de tokens o la aritmética de punteros, alguno deja de cuadrar.

Los contrastes que más dicen:

- **La cinta contra el disco.** Dos contenedores sin nada en común —una tira de bloques pensada para el oído, contra una tabla de sectores— dando el mismo texto byte a byte.
- **EGA contra CGA en El Jabato.** Una se compiló con tabla de compresión y la otra sin ninguna, y toda la prosa sale idéntica. Dice bastante más sobre la expansión de tokens que comparar dos tablas iguales.
- **El Amstrad PCW contra el PC en Los templos sagrados.** Una compilación de Z80 para CP/M, cargada en `0x100` y con relleno tras la cabecera, contra una de PC con el orden de bytes contrario: las siete secciones comunes salen idénticas, en las dos partes.

De paso han salido cosas sobre las aventuras: **el Amiga distribuyó la base de datos compilada para Atari**, sin recompilar, en La Aventura Original y en El Jabato.

### Lo que no se lee, y por qué

Casi todo lo que falla comparte una causa, y **no es protección anticopia**. El formato de disco a medida sí lo es en parte, pero eso ya se supera. Lo que bloquea es que **el cargador guarda la base de datos troceada** y la arma en RAM,
así que en el medio nunca está seguida.

Los discos de Amstrad lo dicen sin rodeos. Sí llevan directorio CP/M, y lo que nombra es `JABATO.B00`, `.B01` y `.B02`, o `TEMPLOS1.B00` hasta `.B02`: los bancos del cargador, numerados. Los ficheros están ahí para leerlos, y la base de datos va repartida entre ellos.

La prueba está en la versión de MSX de El Jabato: su cabecera, sus 288 palabras de vocabulario —las mismas que la de PC— y sus 128 tokens decodifican perfectamente. Con cifrado o compresión no leería ninguna de las tres; solo el
texto está en otro sitio.

La excepción son las cintas del Spectrum de El Jabato, que cargan en trozos de 128 bytes y sí transforman los datos. Pero incluso ahí el disco de la misma máquina se lee sin problema.

Queda por hacer en las bases de datos: entender el mapa de carga de esos cargadores —desbloquearía MSX, Amstrad CPC y Commodore 64 de una vez— y leer los formatos `.CAS` de MSX y `.T64` de Commodore.

De las imágenes ya no queda nada pendiente en las máquinas cuyas bases de datos se leen: sale toda ilustración de toda edición alcanzable.
