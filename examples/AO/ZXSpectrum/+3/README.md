# La Aventura Original para un Spectrum +3

Dos discos, uno por parte, arrancables desde el menú del propio +3. Cada uno
lleva el cargador, el intérprete, la base de datos y las imágenes como ficheros
sueltos.

Lo de aquí supone que `qdaad` y `qpack` están en el PATH. Para la procedencia
de los fuentes y qué llevan corregido a mano, [el README de arriba](../../README.md).

## Construir

```sh
# Parte 1
qdaad compile -i part1/game.sce -o build/p3-part1 -f build -m zx -s plus3 -v 3 --part 1
qpack -f dsk -o AO1 --name AO1 build/p3-part1     # AO1.dsk

# Parte 2
qdaad compile -i part2/game.sce -o build/p3-part2 -f build -m zx -s plus3 -v 3 --part 2
qpack -f dsk -o AO2 --name AO2 build/p3-part2     # AO2.dsk
```

El cargador tiene que llamarse `DISK` y sin extensión, que es lo que el +3
busca al arrancar un disco desde su menú. De eso se encarga `qpack`.

## Las imágenes

Aquí caben todas: **41** en la primera parte y **69** en la segunda. Cada una
viaja como un `.128` propio —cinco bytes de geometría y la pantalla comprimida
con ZX0— y se corta por su última fila con píxeles, no por una altura fija,
que es lo que dejaba sin barra inferior al marco de la presentación.

## La pantalla de presentación

Está en `partN/gfx/loading.scr` y **todavía no viaja en el disco**: la lleva el
cargador BASIC en la cinta de 48K y aquí está por hacer.

## Lo que viene

Este es el árbol donde el +3 se separa de la cinta. La idea es traerle las
descripciones largas de las ediciones de Amiga y de CPC, que en el Spectrum no
cabían y aquí sí. Hoy los fuentes son los mismos que los de 48K salvo en
cuántas imágenes lleva cada uno.

## La parte 2 pide clave

Es del original: al empezar pregunta «¿Clave de acceso?», y la palabra la daba
la primera parte al terminarla.
