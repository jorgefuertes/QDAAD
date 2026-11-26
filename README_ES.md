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

Cuatro de estas comparan nuestra salida con la de las herramientas de referencia. La referencia es DAAD Reborn, de Uto — ver [Agradecimientos](#agradecimientos).

| `make` | | |
|---|---|---|
| `decomp-all` | descompila todas las ediciones y las contrasta entre sí | **38 parejas** |
| `recompile-all` | le pasa cada descompilación al compilador de referencia | **66 aceptadas** |
| `compile-all` | convierte cada edición a la versión 3 y la vuelve a compilar | **66 compiladas, 0 fallos** |
| `compare-drf` | lee cada fuente con los dos analizadores y compara lo que entendió cada uno | **70 iguales, 0 distintos** |
| `compare-drb` | enlaza cada fuente con los dos generadores y compara las bases de datos y el .XMB | **199 iguales, 0 distintos** |
| `compare-daadmaker` | empaqueta cada fuente en una cinta de 48K con los dos empaquetadores | **63 iguales, 0 distintos** |
| `compare-zx0` | comprime cada pantalla con el compresor ZX0 de Einar Saukas y la lee de vuelta | **1450 bloques, 0 distintos** |

`compare-drb` cubre tres destinos por fuente, para que el orden de bytes y el relleno queden ejercitados. Lo que salta es lo que la referencia no enlaza; lo que salta `compare-daadmaker` son los fuentes en idiomas que su compilador no habla y los que no caben en un Spectrum de 48K.

**Ninguno de los siete corre en un clon.** Todos leen `work/`, que son los juegos originales y los checkouts de las propias herramientas de referencia: gigabytes que no son nuestros y que no están en este repositorio. Así que son cosa de ejecutar aquí, con `work/` puesto, y lo que corre en cada empujón es el resto:

| | |
|---|---|
| `make lint` | los linters y `govulncheck` |
| `make test` | la batería, que deja dicho en el resumen cuántos tests saltó por falta de `work/` — unos setenta |
| `make qtest` | las dos partes de La Aventura Original jugadas enteras en dos árboles, y los xmensajes |
| `make example-ao` | las dos cintas de 48K y los cuatro discos de +3, guardados como artefactos de la ejecución |

El último es el que de verdad merece la pena. Las imágenes están en el repositorio —151 pantallas, las tipografías y las pantallas de carga—, así que un clon construye los mismos seis ficheros que esta máquina, y de cualquier commit se puede bajar la cinta o el disco y cargarlo en un emulador sin compilar nada.

## Publicaciones

`make dist` construye las cuatro herramientas para seis máquinas y deja cada juego en un archivo, con un fichero de sumas al lado:

```text
QDAAD_v1.0.1_Linux_amd64.tar.gz    QDAAD_v1.0.1_Windows_amd64.zip
QDAAD_v1.0.1_Linux_arm64.tar.gz    QDAAD_v1.0.1_Windows_arm64.zip
QDAAD_v1.0.1_OSX_amd64.tar.gz      QDAAD_v1.0.1_checksums.txt
QDAAD_v1.0.1_OSX_arm64.tar.gz
```

```sh
sha256sum -c QDAAD_v1.0.1_checksums.txt   # shasum -a 256 -c en un Mac
```

La versión es el tag y nada más. Da nombre al archivo y va enlazada dentro de los cuatro binarios, así que `qdaad --version` responde lo mismo que decía la descarga: no hay ningún número escrito en el fuente del que pueda separarse. Una compilación hecha en un árbol sin tags dice `dev`, que es lo que es.

Subir un tag es lo que hace una publicación, y lo único que la hace: el workflow ejecuta `make dist`, se niega a seguir si los binarios no dicen lo mismo que el tag, y publica los archivos con unas notas generadas a partir de lo que cambió desde el tag anterior.

`compare-zx0` es el único que no pide bytes iguales, y el único cuya referencia no es de Uto. Cada imagen que lleva un Spectrum va comprimida con ZX0, y lo que se exige es que **sus** flujos se lean con **nuestro** descompresor. Es la única manera de distinguir una lectura correcta del formato de una que sólo es consistente consigo misma: la ida y vuelta no lo distingue, porque un compresor y un descompresor que entienden mal lo mismo se entienden entre ellos, y un desplazamiento mal codificado no es una imagen que se vea mal, es una imagen que manda al intérprete a paseo por la memoria. También resuelve cuál de los dos modos de ZX0 es el que lee el intérprete. El fuente de su compresor viaja junto al intérprete, así que el oráculo está en el árbol; ver [internal/zx0](internal/zx0/zx0.go).

No se piden bytes iguales porque dos compresores de un mismo formato son dos programas — pero los tamaños ya coinciden. El nuestro elegía de forma ávida al principio, y esta comparación midió lo que costaba: una décima parte por encima del suyo, y un tercio en el peor bloque. Así que la elección se reescribió como el mismo camino más corto que recorre él, y los dos empatan al byte en los 1450 bloques. Eso se informa, no se exige.

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

### Los dibujos de las ediciones de 8 bits

Las aventuras españolas de 1989 no llevaban mapas de bits. Un dibujo allí es un programa: una tira de órdenes que un motor ejecuta para trazarlo, con una tabla plana de punteros en la cola del bloque diciendo dónde empieza cada uno. Nada marca el bloque, así que se encuentra probando cada desplazamiento como su final y dejando que la disposición lo acepte o lo rechace: una tabla ordenada, que se cierra a sí misma y se acota a sí misma; un registro por dibujo que cuadre con ella al byte; y después, cada dibujo teniendo que llegar a su retorno justo donde empieza el siguiente.

Los dos motores están leídos, y son el mismo diseño dos veces. El del Spectrum pone el opcode en los tres bits bajos, traza con el `DRAW` de la ROM y colorea por atributos; **el del Amstrad lo pone en los bits 1 a 3**, llama al firmware y colorea con cuatro plumas y una paleta de 27 tintas que fija el registro de cada dibujo. Su bloque acaba en `0x9E00` y no en el techo de la memoria, porque de `0xC000` para arriba está la pantalla, y su lienzo lo ancla un `GRA SET ORIGIN 0,32` que el intérprete hace antes de que se dibuje nada — que es lo que hace que el marco con el que abre cada escena caiga en las catorce filas de carácter que pide su registro.

Cada dibujo sale como listado, como PNG y como la pantalla tal cual la tenía la máquina: 6912 bytes para un Spectrum, un modo 1 y sus cuatro tintas para un Amstrad. El formato entero está en [09-graficos.md](docs/DAAD_v2_v3_Research/09-graficos.md) §9.

Ahora mismo se convierten **1798 imágenes**: tipografías, portadas e ilustraciones, en todas las máquinas cuyas bases de datos se pueden leer.

## qdaad y qpack, el camino de vuelta

```sh
go build -o dist/qdaad ./cmd/qdaad
go build -o dist/qpack ./cmd/qpack

qdaad compile -i game.sce -o build/ -f build -m zx -s 48k -v 3 --part 1
qpack -f tap -o GAME --name GAME build/      # GAME.tap
```

Las dos eran una sola orden y se separaron porque son dos trabajos. Compilar va de un fuente y da una base de datos; empaquetar va de una **release**, que va de una máquina. `qpack` escribe `tap`, `tzx`, `dsk`, `dir` —todo lo que la máquina necesita como ficheros sueltos—, `cdt` para una cinta de Amstrad, y `qrun`, nuestro propio contenedor.

Lo que entra en una release además de la base de datos lo declara el fuente, así que el mismo directorio de compilación sirve para cualquiera de ellos:

```text
; qdaad: charset=gfx/chr/udg.chr    la tipografía, que la versión 3 no trae
; qdaad: images=gfx/scr             un directorio de pantallas de Spectrum
; qdaad: loading=gfx/loading.scr    la imagen que se enseña mientras carga el resto
```

Una **cinta de 48K** es un solo flujo: un cargador en BASIC que aparta RAMTOP, y detrás la pantalla de carga, el intérprete, la base de datos, el bloque de gráficos y tantas imágenes como quepan en la memoria que queda. El cargador está escrito y no prestado, y son los mismos bytes que escribe daadmaker, hasta el espacio sobrante de una de sus líneas —que es lo que hace que `compare-daadmaker` signifique algo—. Un `tzx` son los mismos bloques envueltos, 46 bytes más gordo.

Una **cinta de 128K** es esa misma cinta con los bancos de encima de `0xC000` puestos a trabajar, y es la única versión que se queda con lo mejor de los dos lados: una cinta que repartir y el sitio de una máquina de disco. Los cuatro bloques de debajo de la ventana son los del 48K, en las mismas direcciones, y detrás va un bloque **sin cabecera** por cada banco con algo dentro. Sin cabecera porque no los carga el BASIC: el cargador carga lo que vive debajo de `0xC000` y salta, y es el intérprete el que lee el resto por su cuenta y mete cada bloque en el banco que diga el índice.

Primero entran los xmensajes, cada trozo al principio de su banco, porque el desplazamiento que lleva un `XMES` es una posición sobre todos ellos puestos en fila: el intérprete divide por el tamaño del banco para saber cuál es. Después las imágenes, de mayor a menor, a lo que quede de los bancos 3, 4, 6 y 7, luego al banco 1 detrás de los 512 bytes del RAMSAVE y los 6912 del búfer de imagen, y al final a lo que la base de datos haya dejado del banco 0. Que es toda la diferencia que supone: la segunda parte del ejemplo lleva 41 dibujos en una cinta, donde la cinta de 48K de la misma aventura lleva cuatro, y además lleva xmensajes, que una cinta de 48K no puede tener de ninguna manera porque no hay sistema de ficheros de donde leerlos.

Una imagen aquí es el mismo fichero que lleva un +3 —`SCRMAKER` escribe los dos con un solo procedimiento, cinco bytes de geometría y la pantalla recortada como un único flujo ZX0—, así que hay un formato de imagen para las dos máquinas.

Este destino estuvo un tiempo rechazado por una lectura del revés que vale la pena contar. DAAD READY retiró un destino de 128K en la versión B, y era el que se apoyaba en ZXDAAD128, al que llamaba `EXPERIMENTAL_ZX128K`; el destino `ZX128K` se quedó y le pusieron un intérprete nuevo derivado del fuente original de DAAD. Es `DS128KE3.BIN`, que va embebido aquí, y el 3 del final es la versión para la que sirve.

Un **disco de +3** son los ficheros sueltos, y va escrito en el formato propio del +3 y no en el DATA del CPC, que es el que sigue llevando un disco de CPC. La diferencia son dieciséis bytes en el primer sector de la pista 0 —la especificación del disco— y la razón es el byte que hace 7 de ella: cuántos bloques ocupa el directorio. Un disco DATA no lleva especificación, así que la respuesta son 64 nombres y no hay discusión; diciendo cuatro bloques en vez de dos son 128, por dos kilobytes de un disco al que le sobran ochenta. Una imagen es un fichero y un fichero es un nombre, así que en una aventura con un dibujo por localidad lo que se acaba antes son los nombres.

La especificación es la del disco de la propia Aventura Original, leída de él, con ese byte cambiado. Verificado en un +3: el menú carga el disco, un `CAT` saca todos los nombres, y el espacio libre que dice es la aritmética de los campos que se movieron y no la de sus valores por omisión.

Un **disco de CPC** es el intérprete, la base de datos y el bloque de gráficos pegados en un binario a la manera de MCRF, y detrás las imágenes, una por fichero: el intérprete las abre por nombre cuando un `PICTURE` pide una. Son ellas las que se quedan sin sitio, porque en esta máquina no hay compresión ninguna —el intérprete lee una imagen directamente a la memoria de vídeo—, así que un dibujo de catorce filas de carácter son dieciséis kilobytes de un disco con ciento setenta y ocho. `--lines` es donde se negocia eso, y es el `IMGLINES` de DAAD con otro nombre: doce filas reducen a la mitad cada imagen, porque doce es lo máximo que deja compartir un registro de 2K a dos bandas de una misma imagen. Lo que se queda fuera se nombra, no se tira en silencio.

Pedirle pantalla de carga a una release de CPC la convierte de un fichero en tres —el cargador, la imagen y el juego—, porque enseñar algo antes de que arranque el intérprete exige un programa que corra antes que él. El cargador pone el modo y la paleta, sube la imagen y carga el juego por debajo. Lo único que no se deduce de ningún papel son sus cuatro primeras instrucciones: `RUN` entra en un binario por MC START PROGRAM, que reinicia el firmware y se lleva por delante los parches de AMSDOS, así que un cargador que simplemente abra un fichero recibe un «Press PLAY» con el disco metido. Recorrer las ROM de fondo, preguntar cuál responde a `DISC`, reiniciar con todas activas y después inicializar esa por número es lo que lo arregla — y es la secuencia del propio intérprete, leída en `0x2458`.

El cargador de cinta no lleva nada de eso, y el motivo es la misma secuencia leída al revés: el paseo por las ROM, él solo, devuelve AMSDOS en cualquier máquina que tenga unidad, y a partir de ahí cada fichero que el cargador pide se le pide a un disco que no está. Tal como lo deja MC START PROGRAM, el firmware ya está leyendo la cinta por la que entró el `RUN`, que es todo lo que una cinta necesita.

Una **cinta de CPC** es un `cdt`, y no es la cinta del Spectrum envuelta: el Amstrad tiene formato propio —un fichero troceado en bloques de 2K, cada bloque escrito como un registro de cabecera y otro de datos, y cada registro en segmentos de 256 con un CRC-16/CCITT detrás de cada uno—. Los pulsos y la suma están leídos de la cinta de Amstrad de la propia Aventura Original. Y lo que hace con las imágenes de la aventura es la otra mitad. A una cinta no se le puede preguntar en cualquier orden, así que no van en ella: van **a memoria junto con el juego**, comprimidas con ZX0, y dos parches de tres bytes apuntan al intérprete hacia ellas en vez de hacia un fichero. Así una imagen baja a unos 1300 bytes, y lo que queda libre entre el final del juego y el bloque de gráficos del intérprete da para una docena larga; las demás se nombran. El intérprete ya pregunta si la imagen está antes de pintarla y ya acepta un no por respuesta, así que una que no quepa sale como ninguna imagen y no como un cuelgue — y cuando no cabe ninguna, se le quita la pregunta entera para que la cinta no vuelva a parar a pedir casete.

`--ram 128` es donde la docena se convierte en todas, tanto en cinta como en disco. Los cuatro bancos libres de un 6128 son 64K que el intérprete no toca nunca —no escribe en más puertos que los del PPI y deja en paz la configuración de memoria de la puerta lógica de principio a fin—, así que ahí van las imágenes, llenando un banco tras otro, y el cargador conmuta cada uno a la ventana de `0x4000` según lo lee. Las 28 imágenes de la primera parte de La Aventura Original caben con sitio de sobra. Vale tanto para disco como para cinta, y `--ram 128` lo dice en los dos: un disco de 128K se lee una vez y no se vuelve a tocar, mientras que uno de 64K abre un fichero cada vez que el jugador entra en una localidad. La release no arranca en un 464 ni en un 664, y ese es todo el precio.

Una **release de PC** no es un contenedor, y es la única de aquí que no lo es. Todas las demás acaban en algo con forma —un flujo de bloques en direcciones fijas, un formato de disco con directorio y especificación— porque todas las demás tienen que meter una aventura en una máquina que no tiene sitio. Un PC tiene sistema de ficheros y megabytes de él, así que la respuesta de DAAD son ficheros sueltos con los nombres que abre el intérprete, y todo el empaquetado consiste en poner los bytes correctos en el nombre correcto:

```text
PCDAAD.EXE  el intérprete
DAAD.DDB    la aventura
DAAD.FNT    la tipografía, con una anchura por glifo
DAAD.MSD    la imagen que se ve antes que nada
NNN.MSD     una imagen
0.XMB       los textos que se leen de disco
PRUEBA.BAT  una línea, para que haya algo evidente que teclear
```

Esos nombres no son una convención leída de un script de construcción. **El fuente de PCDAAD está en el repositorio** —veintinueve ficheros de Pascal—, así que lo que abre está leído y no medido: `DAAD.DDB` es la base de datos que toma por omisión, `DAAD.FNT` se prueba antes que `DAAD.CHR`, y una imagen son tres dígitos y `.MSD` antes de recurrir a `.pcx` y a `.VGA`. Leer gana a medir, y este destino es el único sitio de este proyecto donde el otro lado entero estaba para leerlo.

Lo que no va en la release es DOSBox. La de DAAD READY lleva `dosbox.exe`, dos DLL de SDL y un `.BAT` que los arranca, que es un programa de Windows para ejecutar un programa de DOS; se monta el directorio en el DOS que cada uno tenga y se teclea el nombre.

La tipografía es lo único que un PC hace distinto de todas las demás máquinas de aquí: **su texto es proporcional**. Un `.FNT` es el formato de SINTAC y lleva una anchura por cada uno de los 256 glifos en sus últimos 2304 bytes, así que una `i` mide tres píxeles y una `m` seis. Un fuente que declare una tipografía corriente de 8 bits, de 2048 bytes, la recibe como `DAAD.CHR` y el intérprete le da seis de ancho a todos los glifos —que es lo que hacen las máquinas de 8 bits de todos modos—.

Una imagen es un PCX, porque es lo que toman las herramientas de DOS de DAAD y por tanto lo que tiene en disco quien dibuje para él. Es el único destino cuyas imágenes no son un volcado de una memoria de vídeo, y por eso sus fuentes llevan `000.pcx` donde los demás llevan `000.scr`. Lo que va en la release es `.MSD`, el formato propio de SCRMAKER: 16 bytes de geometría, la paleta entera de 768, y un byte por píxel sin compresión ninguna —31504 bytes para un dibujo de 320×96, donde un Spectrum mete cuarenta en 16K—. Iba a un disco duro y a nadie le importó.

La parte que hay que saber es la paleta. DAAD se reserva las dieciséis primeras entradas, porque una release de PC elige sus tintas con los mismos números que una de 8 bits e `INK 6` tiene que ser amarillo. Así que cada píxel del dibujo sube dieciséis puestos y su paleta sube con él, y las dieciséis del Spectrum se sientan abajo. Un dibujo que use las dieciséis últimas entradas no tiene a dónde subir y se rechaza en vez de darle la vuelta, porque darle la vuelta pintaría esos píxeles con los colores del texto y quedaría casi bien.

Ese desplazamiento es para lo que está `make compare-scrmaker`: compila SCRMAKER de su propio fuente, convierte los mismos PCX y compara los bytes. Si el desplazamiento está mal, la imagen se pinta igual, con los colores cambiados y sin un error en ninguna parte.

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

## Agradecimientos

**Uto**, por DAAD Reborn y por los años de ingeniería inversa que vinieron antes. Casi todo lo que está escrito sobre este formato está escrito porque él lo averiguó primero, y ha seguido contestando preguntas y compartiendo lo que sabe mientras esto se construía — información y criterio, que es la mitad más difícil de dar.

Los tres arneses de comparación sostienen nuestra salida contra la de DAAD Reborn byte a byte, y esa comparación vale precisamente porque las dos implementaciones son independientes: el código de aquí se escribió desde el formato y desde las bases de datos, no desde el suyo. Entender el formato es otra cosa, y buena parte de ese entendimiento es suyo.

**Natalia Pujol**, por msx2daad, que es DAAD escrito en C y es lo que el intérprete de `internal/vm` sigue condacto a condacto.

**Aventuras AD** —Andrés Samudio, Manuel González, Carlos Marqués y quienes trabajaron con ellos— por las cinco aventuras a las que va dedicado todo esto, y por un formato lo bastante cuidado como para que todavía se pueda leer treinta y siete años después.
