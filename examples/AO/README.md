# La Aventura Original, portada a un intérprete de versión 3

La Aventura Original es de Aventuras AD, 1989 —dirección de Andrés Samudio,
programación de Manuel González, gráficos de Carlos Marqués—. Lo que hay aquí
no es la aventura tal y como se publicó: es su base de datos descompilada y
después **convertida** para que la ejecute un intérprete de versión 3, que es
otra cosa y conviene decirlo en la primera línea.

Sirve de ejemplo grande de verdad —49 localidades y 41 dibujos la primera
parte, 109 y 69 la segunda— y de banco de pruebas: casi todo lo que sabemos de
la conversión de versión 1 a versión 3 se ha aprendido arreglando esto. Lo que
cada diferencia es y por qué, está en
[07-daad-v3.md](../../docs/DAAD_v2_v3_Research/07-daad-v3.md) §11.

```
ZXSpectrum/48/    README.md, part1/, part2/     dos cintas
ZXSpectrum/+3/    README.md, part1/, part2/     dos discos
```

Las órdenes de cada máquina están en su propio README. `make example-ao`
construye las cuatro de una vez.

## De dónde sale cada fichero

Los `.sce` y el `gfx/` los escribe el descompilador desde la cinta original:

```
qundaad decompile --v3 \
  --input "La Aventura Original - Part 1.tzx" \
  --output <directorio>
```

Pero **no salen así tal cual**. Lo que hay aquí lleva correcciones a mano, y
mientras el descompilador no las sepa hacer solo, esta copia es la buena. Las
que hay hoy:

- **`game.sce`** declara el juego de caracteres, las imágenes y la pantalla de
  presentación:

  ```
  ; qdaad: charset=gfx/chr/udg.chr
  ; qdaad: images=gfx/scr
  ; qdaad: loading=gfx/loading.scr
  ```

  El juego de caracteres de 1989 no es el que carga un intérprete v3, y se
  nota: los dibujitos que van al lado del nombre de un objeto salen como
  itálicas si no se carga.

- **`v3-loop.sce`** es el bucle del juego, que una base de formato 1 no trae
  porque en 1989 vivía dentro del intérprete. El que escribe el descompilador
  todavía es el de «aquí no hay ilustraciones»; éste dibuja. Lleva además:

  - `CLEAR 40` al limpiar y `SET 40` detrás del `DISPLAY 0`. La bandera 40 es
    como el intérprete del Spectrum le decía a la aventura si la localidad
    tenía dibujo; ninguna de las dos aventuras la escribe, sólo la leen. Sin
    reponerla, la imagen genérica tapa todas las localidades.
  - `WINSIZE 24 42` y no el `25 127` de costumbre. Un Spectrum tiene 24 filas,
    y pedir 25 manda la primera línea a una fila que no existe.
  - El texto en la fila 14, que es lo que mide la mayoría de las imágenes.
  - `NOTAT 0` en las entradas que encogen la ventana, por la pantalla de
    arranque: la localidad 0 pregunta con `PRINTAT 11 21` sobre la pantalla
    entera, y con la ventana ya encogida esa fila no existe.

- **`gfx/loading.scr`** es la pantalla de presentación de la cinta original, el
  bloque `AO$` de 6912 bytes. Es la misma en las dos partes.

El resto de los `.sce`, el `udg.chr` y los `.scr` salen del descompilador sin
tocar. Fuera quedan los `.png` y los `.plot`: son derivados para mirar, no para
construir.

## Las dos máquinas

Son dos árboles y no uno con dos destinos, porque van a separarse de verdad: la
idea para el +3 es traerle las descripciones largas de las ediciones de Amiga o
CPC, que en el Spectrum no cabían.

Hoy la única diferencia es **cuántos dibujos lleva cada uno**.

| | Parte 1 | Parte 2 |
|---|---|---|
| Dibujos que tiene | 41 | 69 |
| En la cinta de 48K | 4 | 3 |
| En el disco del +3 | 41 | 60 de 69 |

En la cinta entran unas 10, así que cada parte lleva las que hacen falta para
que el juego se entienda y de las demás se encarga la genérica, que es lo que
hace la aventura. En el disco de la segunda parte se quedan fuera 9 por tamaño;
`qpack` dice cuáles y no corta ninguna por la mitad.

## Lo que todavía no está bien

- **Un punto suelto** aparece en la ventana de texto en algunas localidades. No
  es ningún mensaje de la base —se descartaron los 63 de sistema y los 140 de
  usuario marcándolos todos— y se coloca dentro de la ventana de texto, no en
  una fila fija de la pantalla.
- **La pantalla de presentación no viaja en el disco del +3**. En la cinta sí.
- **El descompilador no reproduce esta copia**: todo lo de la lista de arriba
  hay que ponerlo a mano después de descompilar.
