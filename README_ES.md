# QDAAD

Reimplementación en Go del sistema DAAD de creación de aventuras conversacionales.

El proyecto tiene dos mitades. `qdaad` compilará fuentes `.SCE`, además de un formato propio más moderno, a base de datos `.DDB`, y está en construcción. **`qundaad` ya funciona**: lee una base de datos DAAD y devuelve el fuente que la produjo.

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

La salida es UTF-8, legible y modificable hoy, repartida en un fichero por sección y unida con `#INCLUDE` desde `game.sce` — que es como se organizaban los fuentes de la época: el manual de 1991 avisa de los peligros de «incluir un fichero TOK que contenga otro `/TOK`», advertencia que solo tiene sentido si trocear era lo normal.

Junto al fuente van dos directorios más: `chr/` con las tipografías y `gfx/` con las imágenes. De cada cosa se guarda **el binario tal como venía y su conversión a PNG**, y de los archivos de ilustraciones también cada dibujo por separado, con un índice que dice qué localidades usan cada uno y con qué paleta. Con `--no-binaries` se dejan solo las conversiones, que ahorra la mitad del espacio.

### Lo que deduce en vez de suponer

Una `.DDB` no declara casi nada de lo que hace falta para leerla. `qundaad` no se fía de tablas indexadas por plataforma —de ahí que las herramientas de referencia den una base de datos de PC por *little-endian* cuando no lo es— y lo saca de los propios datos:

- **Orden de bytes y tamaño de cabecera**, probando las cuatro combinaciones y quedándose con la que hace que la cabecera se describa a sí misma. En estas cinco aventuras aparecen las cuatro.
- **La dirección de carga**, cuando la base de datos no es un fichero sino que está incrustada en un programa o puesta a pelo sobre un disco. Sus punteros son direcciones absolutas, y el vocabulario, que va pegado a la cabecera, delata dónde se cargó todo. - **Si hay tabla de compresión**, que es opcional: la versión CGA de El Jabato se compiló sin ninguna.

### Soportes que abre

`internal/media` lee el contenedor de cada máquina, porque una base de datos casi nunca es un tramo contiguo de bytes en el medio original:

| Soporte | |
|---|---|
| Amiga `.ADF` | OFS (con cabecera en cada bloque de datos) y FFS |
| Atari ST y MSX `.ST`, `.DSK` | FAT12 |
| Commodore `.d64` | CBM DOS, sectores encadenados |
| Amstrad y Spectrum `.DSK` | CPCEMU, variante original y extendida |
| Cintas `.TZX` y `.CDT` | los 25 tipos de bloque del formato |
| Commodore `.TAP` | no: guarda los pulsos, no los bytes |

Varios de estos discos **no llevan sistema de ficheros**: se formatearon para el cargador del propio juego. Ahí no hay directorio que recorrer, así que se buscan las bases de datos por firma entre los sectores y se las obliga a demostrar que lo son —cabecera coherente, tablas de texto y conexiones que leen, y prosa que parece prosa— antes de aceptarlas.

## Las imágenes y las tipografías

Las tipografías son 2176 bytes: **cabecera AMSDOS de 128 y 256 glifos de ocho filas**. Que lleven la cabecera de Amstrad hasta en la versión de PC dice de dónde salió la herramienta — Aventuras AD trabajaba en CPC y PCW. Las letras se dibujan en seis columnas de las ocho; los glifos del 128 en adelante no son letras sino **tramas de sombreado**, y esas sí usan las ocho.

De ahí sale además la confirmación del juego de caracteres de DAAD leída en los propios bitmaps, no en el código de otras herramientas: 16 = ª, 17 = ¡, 18 = ¿, 19 = «, 20 = », 21 a 25 = á é í ó ú, 26 = ñ, 27 = Ñ, 28 = ç, 29 = Ç, 30 = ü, 31 = Ü.

Las **portadas** se dibujan en las cuatro rutas: CGA modo 4 con sus bancos entrelazados, EGA modo 0Dh en cuatro planos, y Degas de 68000 — que en Atari entrelaza los planos por palabra y en Amiga los guarda enteros, cosa que el fichero no dice y hay que tomar del soporte.

Hay **tres compresiones**, una por generación, y ninguna es un formato conocido. La del PC salió del propio compresor: `DMG.EXE`, el gestor de gráficos que construyó estos archivos, sigue viniendo con la entrega de 1991, y su rutina en `0x3b4f` cuenta cuántas veces sale cada valor de byte en la imagen, se queda con los cuatro más frecuentes y escribe la imagen byte a byte — salvo que un byte de esos cuatro va seguido de la cuenta de cuántas veces se repite. Cuántos de los cuatro reciben ese trato se decide probando los cuatro y quedándose con el que menos ocupa. Al leerla, 189 de las 191 comprimidas dan exactamente el tamaño que dice su cabecera habiéndose comido el flujo entero.

Las otras dos son las de 68000, y ninguna cede a ningún esquema estándar: cada píxel son cuatro bits, y la cabecera lleva **una máscara de dieciséis bits que dice qué colores van seguidos de una repetición**. Como el significado de los cuatro bits siguientes depende del color recién leído, no hay esquema estándar que encaje. Salió leyendo los intérpretes con `objdump -m m68k`: la rutina `0x20aa` del Atari de La Aventura Original, y la `0x269a` de Los Templos Sagrados.

Hay dos generaciones de archivo y el fichero no dice cuál es. Las aventuras tardías ensancharon la ranura de 44 a 48 bytes y cambiaron de dónde se sacan los píxeles —un nibble de un longword en vez de un bit de cada uno de cuatro bytes—, y sus ediciones de PC llevan el mismo archivo con los bytes al revés. Se prueban las formas y se queda la que cuadra: un archivo pone su primera imagen justo donde acaba su tabla de ranuras, y solo la lectura correcta la deja ahí.

Ahora mismo se convierten **1795 imágenes**: tipografías, portadas e ilustraciones, en todas las máquinas cuyas bases de datos se pueden leer.

## Con qué se ha probado

Las cinco aventuras de Aventuras AD, en todas las ediciones que se han conseguido: **81 ficheros de soporte, de los que salen 49 bases de datos**.

| Aventura | PC | Amiga | Atari ST | Amstrad CPC | ZX Spectrum | C64 | MSX | PCW |
|----------|----|-------|----------|-------------|-------------|-----|-----|-----|
| La Aventura Original | sí | sí | sí | sí | sí | no | — | — |
| El Jabato | sí | sí | 1/2 | no | 1/3 | no | no | — |
| Cozumel | sí | sí | sí | sí | sí | no | no | no |
| Chichén Itzá | sí | sí | sí | no | no | no | no | — |
| Los templos sagrados | sí | sí | sí | no | no | no | no | no |

(«1/2» en el Atari ST de El Jabato es el segundo disco, que solo trae gráficos.)

### Cómo se comprueba que el resultado es correcto

`make decomp-check` descompila las cinco aventuras y contrasta las ediciones entre sí: **29 emparejamientos**, doce que tienen que dar el mismo fuente entero y diecisiete que tienen que coincidir en el texto y los datos.

Es una comprobación fuerte porque los binarios comparados no se parecen en nada —distinto orden de bytes, distinto tamaño, distintos desplazamientos— y aun así tienen que converger en el mismo texto. Si se rompe la deducción del orden de bytes, la tabla de tokens o la aritmética de punteros, alguno deja de cuadrar.

Los dos contrastes que más dicen:

- **La cinta contra el disco.** Dos contenedores sin nada en común —una tira de bloques pensada para el oído, contra una tabla de sectores— dando el mismo texto byte a byte.
- **EGA contra CGA en El Jabato.** Una se compiló con tabla de compresión y la otra sin ninguna, y toda la prosa sale idéntica. Dice bastante más sobre la expansión de tokens que comparar dos tablas iguales.

De paso han salido cosas sobre las aventuras: **el Amiga distribuyó la base de datos compilada para Atari**, sin recompilar, en La Aventura Original y en El Jabato.

### Lo que no se lee, y por qué

Casi todo lo que falla comparte una causa, y **no es protección anticopia**. El formato de disco a medida sí lo es en parte, pero eso ya se supera. Lo que bloquea es que **el cargador guarda la base de datos troceada** y la arma en RAM,
así que en el medio nunca está seguida.

La prueba está en la versión de MSX de El Jabato: su cabecera, sus 288 palabras de vocabulario —las mismas que la de PC— y sus 128 tokens decodifican perfectamente. Con cifrado o compresión no leería ninguna de las tres; solo el
texto está en otro sitio.

La excepción son las cintas del Spectrum de El Jabato, que cargan en trozos de 128 bytes y sí transforman los datos. Pero incluso ahí el disco de la misma máquina se lee sin problema.

Queda por hacer en las bases de datos: entender el mapa de carga de esos cargadores —desbloquearía MSX, Amstrad CPC y Commodore 64 de una vez— y leer los formatos `.CAS` de MSX, `.T64` de Commodore y los discos de PCW.

De las imágenes ya no queda nada pendiente en las máquinas cuyas bases de datos se leen: sale toda ilustración de toda edición alcanzable.

## Estructura

```
cmd/qundaad/       el descompilador y su CLI
cmd/qdaad/         el compilador, en construcción
internal/ddb/      el modelo de datos: vocabulario, objetos, mensajes, procesos
internal/media/    lectura de discos y cintas de cada máquina
docs/              investigación sobre el formato y el manual
work/              material de referencia, no forma parte del programa
```

## Desarrollo

```sh
make test           # go vet y la suite completa
make lint           # gofumpt, staticcheck, golangci-lint, govulncheck
make decomp-check   # descompila las cinco aventuras y las contrasta
make dead           # código sin usar
```
