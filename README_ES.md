# qDAAD

Reimplementación en Go del sistema DAAD de creación de aventuras conversacionales.

El proyecto tiene dos mitades. `qdaad` compilará fuentes `.SCE`, además de un formato propio más moderno, a base de datos `.DDB`, y está en construcción. **`qundaad` ya funciona**: lee una base de datos DAAD y devuelve el fuente que la produjo.

## qundaad, el descompilador

```sh
go build -o dist/qundaad cmd/qundaad/qundaad.go

qundaad decompile --input PART1.DDB              --output src/
qundaad decompile --input ORIGINAL.ADF           --output src/
qundaad decompile --input "Aventura Original.dsk" --output src/
```

La entrada puede ser la base de datos suelta o **la imagen del disco o la cinta original**. Una imagen se abre, se recorre y cada base de datos que haya dentro se descompila en su propio directorio.

La salida es UTF-8, legible y modificable hoy, repartida en un fichero por sección y unida con `#INCLUDE` desde `game.sce` — que es como se organizaban los fuentes de la época: el manual de 1991 avisa de los peligros de «incluir un fichero TOK que contenga otro `/TOK`», advertencia que solo tiene sentido si trocear era lo normal.

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

Queda por hacer: entender el mapa de carga de esos cargadores —desbloquearía MSX, Amstrad CPC y Commodore 64 de una vez— y leer los formatos `.CAS` de MSX, `.T64` de Commodore y los discos de PCW.

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
