# La Aventura Original para un Spectrum de 48K

Dos cintas, una por parte. Cada una lleva su cargador BASIC, la pantalla de
presentación, el intérprete, la base de datos, los gráficos y el stub, todo en
un flujo.

Lo de aquí supone que `qdaad` y `qpack` están en el PATH. Para la procedencia
de los fuentes y qué llevan corregido a mano, [el README de arriba](../../README.md).

## Construir

```sh
# Parte 1
qdaad compile -i part1/game.sce -o build/48-part1 -f build -m zx -s 48k -v 3 --part 1
qpack -f tap -o AO1 --name AO1 build/48-part1     # AO1.tap
qpack -f tzx -o AO1 --name AO1 build/48-part1     # AO1.tzx

# Parte 2
qdaad compile -i part2/game.sce -o build/48-part2 -f build -m zx -s 48k -v 3 --part 2
qpack -f tap -o AO2 --name AO2 build/48-part2     # AO2.tap
qpack -f tzx -o AO2 --name AO2 build/48-part2     # AO2.tzx
```

El `tap` y el `tzx` llevan los mismos bloques: el segundo es el primero con una
firma y una pausa de un segundo detrás de cada bloque. Carga igual y pesa 46
bytes más.

## Qué hace el cargador

```
10 CLEAR VAL "24575"
20 BORDER 0: PAPER 0: INK 7: CLS
30 LOAD ""CODE                                      la pantalla, a 16384
40 POKE PEEK 23633+256*PEEK 23634,VAL "111"         calla a la ROM
50 LOAD ""CODE:LOAD ""CODE:LOAD ""CODE:LOAD ""CODE  interprete, base, stub, imagenes
60 RANDOMIZE USR VAL "24576"                        y arranca
```

La pantalla va a memoria de vídeo, así que se pinta sola según entra y no hace
falta ni rutina ni copia. El cargador de 1989 la metía en 30000 y la volcaba
con un `RANDOMIZE USR 30000`; esto es lo mismo con menos pasos. El `POKE` es el
truco de siempre para que la ROM no escriba «Bytes:» encima de la imagen, con
la dirección sacada de `CURCHL` en vez de escrita a mano, igual que hacía el
original en su línea 200.

Al cargar el último bloque arranca la aventura, sin tecla de por medio.

## Las imágenes

En una cinta de 48K no caben las 41 de la primera parte ni las 69 de la
segunda: entran unas 10. Así que cada parte lleva sólo las que hacen falta para
que el juego se entienda, y de las demás se encarga la genérica.

| Parte 1 | | Parte 2 | |
|---|---|---|---|
| `003` | la cabaña del bosque | `001` | la presentación, que es también la primera localidad |
| `027` | la genérica | `095` | la genérica |
| `029` | la lápida, al morir | `092` | la lápida |
| `044` | el marco de la presentación | | |

Que la genérica valga para el resto no es un apaño nuestro: es lo que hace la
aventura, y lo decide con la bandera 40 —«esta localidad tiene dibujo»— que el
bucle repone. Si quieres más imágenes, añade sus `.scr` a `partN/gfx/scr/` y
vuelve a empaquetar: `qpack` avisa por su cuenta de las que no quepan y no
corta ninguna por la mitad.

## La parte 2 pide clave

Es del original: al empezar pregunta «¿Clave de acceso?», y la palabra la daba
la primera parte al terminarla.
