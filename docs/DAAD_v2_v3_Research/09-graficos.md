# 09 — Imágenes

Cómo se incluyen imágenes en una aventura DAAD: formatos de entrada por plataforma, propiedades
de una imagen, condactos implicados y empaquetado.

Fuente principal: `daad-ready/DOC/multimedia_en.html` y `multimedia_es.html`, más el código de
`TOOLS/SCRMAKER`, `TOOLS/MALUVA/sc2daad`, `TOOLS/img2daad` y `TOOLS/imgwizard`.

---

## 1. El modelo

**Las imágenes no viven dentro del DDB.** Son ficheros externos, o bloques empaquetados aparte,
numerados de `000` a `255` con 3 dígitos. La plantilla estándar muestra automáticamente la
imagen cuyo número coincide con el de la localidad al entrar en ella:

```text
PICTURE @fPlayer       ; carga la imagen de la localidad actual
DISPLAY @fDarkF        ; la muestra si no hay oscuridad
```

Los recursos de número ≥ 100 se suelen reservar para imágenes invocadas a mano con
`PICTURE n` + `DISPLAY 0`. **No se deben mezclar 2 tipos de recurso con el mismo número.**

### `PICTURE` y `DISPLAY`

- `PICTURE n` — condición: prepara o carga la imagen *n*. Devuelve falso si no existe.
- `DISPLAY 0` — vuelca al pantalla la imagen preparada.
- `DISPLAY n` con n ≠ 0 — limpia la pantalla.

La separación existe porque en las máquinas de 8 bits la carga desde cinta o disco es lenta y
conviene decidir *después* si hay que pintar.

---

## 2. Propiedades de una imagen

Cuatro atributos ortogonales, no soportados por todas las plataformas:

| Propiedad | Significado |
|---|---|
| **Fija** | Se dibuja en una posición absoluta e **ignora `WINDOW`, `WINSIZE` y `WINAT`** |
| **Flotante** | Se dibuja dentro de la ventana activa |
| **Rango parcial de paleta** | Solo aplica un tramo de la paleta al cargarse, preservando los colores del texto |
| **Bufferizada** | Se precarga en memoria para mostrarla sin acceso a disco |

En `SCRMAKER`, la ausencia del par `X,Y` en la línea de órdenes marca la imagen como flotante;
su presencia la fija. El prefijo `L:` en el nombre de salida la marca como bufferizada (solo +3,
con 8960 bytes disponibles).

El flag `/s` desplaza la paleta 16 colores hacia arriba y coloca abajo los 16 colores estándar
del Spectrum, para que el texto conserve sus colores. Es la razón de que se recomienden
**240 colores** en lugar de 256 en PC y ZX Next.

Un rango invertido (`1-0`) significa "no apliques ninguna paleta".

---

## 3. Formatos de entrada por plataforma

| Carpeta | Plataforma | Formato | Restricciones |
|---|---|---|---|
| `IMAGES/nnn.SCR` | ZX 48K/128K/+3/esxDOS | SCR de emulador | **6912 bytes exactos** |
| `IMAGES/nnn.PCX` | ZX Next | PCX indexado 8 bpp | 256×192; 240 colores recomendados; magenta puro `#FF00FF` = transparente |
| `IMAGES/ZXUNO/nnn.SCR` o `.MLT` | ZX-Uno | Timex HiColor 8×1 + ULAplus | SCR extendido de 12352 B, o MLT de 12288/12352; paleta de 64 colores, los 16 primeros para texto |
| `IMAGES/nnn.SC2` | MSX1 | Screen 2 | **14343 bytes exactos**; atributos 8×1 |
| `IMAGES/nnn.SC8` | MSX2 | Screen 8 | 256×212, 256 colores fijos |
| `IMAGES/CPC/nnn.SCR` + `nnn.PAL` | Amstrad CPC | OCP Art Studio | 320×200 modo 1 (4 colores); 160×200 modo 0 con split |
| `IMAGES/nnn.ART` | C64 HiRes | Art Studio | 320×200 |
| `IMAGES/nnn.KOA` o `.KLA` | C64 multicolor | Koala | 160×200, 16 colores, **96 px de alto obligatorio** |
| `IMAGES/nnn.PRG` | Plus/4 | Botticelli HiRes | sin multicolor |
| `IMAGES/nnn.PNG` o `.PI1` | Amiga y Atari ST | PNG o Degas | 320×200, **máximo 16 colores**; paleta de 3 bits por componente |
| `IMAGES/PC/nnn.PCX` | PC/DOS VGA | PCX indexado 256 col | 320×200; 240 colores recomendados |
| `IMAGES/PC/SVGA/nnn.PCX` | PC/DOS SVGA (`HIRES=1`) | PCX indexado 256 col | 640×400 |
| `IMAGES/PC[/SVGA]/nnn.FLI` | PC/DOS | Autodesk Animator FLI | 320×200; se reproduce con `GFX n 13/14` |
| `IMAGES/HTML/nnn.png` | HTML | PNG **RGB**, no indexado | ≤320×200 |
| `IMAGES/HTML/nnn.mp4` | HTML | MP4 | pantalla completa, `GFX n 13/14` |
| `IMAGES/EXPERIMENTAL/16/` | ST/STE experimental | PNG 16 col | 320×200 |
| `IMAGES/EXPERIMENTAL/32/` | Amiga OCS/ECS exp. | PNG 32 col | 320×200 |
| `IMAGES/EXPERIMENTAL/256/` | AGA, Falcon, Win/Mac/Linux | PNG 256 col | 320×200 |
| `IMAGES/EXPERIMENTAL/TRUECOLOR/` | Amiga HAM6 exp. | PNG hasta 4096 col | 320×200 |
| `IMAGES/EXPERIMENTAL/HIRES/` | Win/Mac/Linux con `HIRES=1` | PNG 256 col | 640×400 |

**Pantallas de carga**: `IMAGES/DAAD.SCR` (ZX), `DAAD.SC2` (MSX1), `DAAD.SC8` (MSX2),
`IMAGES/PC[/SVGA]/DAAD.PCX` (**siempre 320×200 aunque el juego sea SVGA**), `IMAGES/DAAD.PNG`
(Amiga y ST, 320×200, ≤16 colores).

La variable `IMGLINES` de `CONFIG.BAT` fija la **altura útil** de las imágenes y afecta a todos
los targets. El valor por defecto es 96.

---

## 4. Los cuatro conversores

### `SCRMAKER` — el conversor multi-target principal

```text
SCRMAKER <target> <entrada> [L:]<salida> <ancho> <alto> [X,Y] [pal_lo-pal_hi] [/s|/SCR]
target ∈ pc | cpc | spectrum | spectrum128 | c64 | zxuno | specnext
```

Salidas: `.ZX` (Spectrum), `.128` (Spectrum 128, desde un SCR comprimido con zx0), `.LY2`
(ZX Next Layer 2), `.ZXU` (ZX-Uno), `.MSD` (PC).

Escribe un `SCRMAKER.LOG` que consume después `plus3cache.php` para construir el fichero de
imágenes bufferizadas del +3.

### `sc2daad` (Maluva) — conversor clásico

```text
SC2DAAD <target> <entrada> <salida> <líneas> [[transparencia|colorBorde] [colorDestino]]
target ∈ c64 | cp4 | cpc | msx | zx | uno | next
```

Solo produce **imágenes fijas en la parte superior**, a ancho completo. Es el camino heredado;
`SCRMAKER` lo sustituye donde hay soporte moderno.

### `img2daad` — Amiga y Atari ST clásicos

PHP con GD. Empaqueta imágenes **y sonidos en la misma pasada** en un fichero `.DAT`:

```text
img2daad.php IMAGES;SOUNDS PART1.DAT [-a] [-c] [-<altura>]
```

`-a` selecciona el modo Amiga con paleta normalizada de 4 bits y firma `0xDAADDAAD`. El ancho
está fijado a 320.

**Formato DAT/DMG** (documentado en `img2daad.php:44-77`): cabecera global con firma `0x0300`,
modo de pantalla, número de imágenes y tamaño; después una tabla de **256 entradas de 48 bytes**
(offset 4 B, flags 2 B, X 2 B —o frecuencia si es un sonido—, Y 2 B, inicio de paleta 1 B, fin de
paleta 1 B, paleta 32 B, firma CGA 4 B); y por último los datos (ancho 2 B con bit 15 =
comprimido, alto 2 B con bit 15 = audio, tamaño 2 B, píxeles). La compresión es RLE con una
máscara de 16 bits que indica qué colores se comprimen.

Cada imagen admite un `.JSON` propio con sus propiedades: `float`, `buffer`, `X`, `Y`, `PCS`,
`PCE`, `width`, `height`, `compress`, y `clone` + `location` para reutilizar la imagen de una
localidad **anterior**.

### `imgwizard` — MSX2

De Natalia Pujol. Convierte a un formato `IMx` propio, por chunks, con magic `"IMG"`:

| Tipo | Chunk |
|---|---|
| 0 | Redirección de localidad |
| 1 | Paleta (32 bytes; ignorado en SC8 y SC12) |
| 17 | ClearWindow |
| 19 | Pause |
| 20 / 21 | **Comando y datos del V9938** |
| 128 | Info: tipo de píxel, de paleta, de chipset |
| 129 | Imagen fija |

Compresores por chunk: `RAW`, `RLE` (por defecto), `PLETTER`, `ZX0`. Chunk máximo de 2043 bytes.
La transparencia se resuelve en 2 pasadas (`LMMC|AND` y `LMMC|OR`) en SC5/6/7/8; SC12 no está
soportado.

Los datos se envían por streaming directo al puerto `#9B` del V9938: el intérprete no mantiene
la imagen en RAM.

---

## 5. Empaquetado por plataforma

| Plataforma | Dónde viven las imágenes |
|---|---|
| ZX 48K | Troceadas por `ZXsplitter.php`, comprimidas con ZX0 y paginadas con `pager48k.php` en `INDEX.BIN` + `PAGE0.TMP` |
| ZX 128K | Comprimidas con ZX0 y repartidas en bancos por `pager128k.php`; ver el mapa de memoria en §6 |
| ZX +3 | Igual, más `DAAD.GRA` con las bufferizadas (`plus3cache.php`) |
| ZX Next, ZX-Uno, esxDOS | Ficheros sueltos en la tarjeta SD |
| MSX1 / MSX2 | Ficheros `.MS2` / `.IMx` dentro del `.DSK` |
| CPC | Empaquetadas por `MCRF` en `DAAD.BIN` junto al intérprete y el DDB |
| C64 / Plus4 | Ficheros dentro del `.D64` |
| PC/DOS | Ficheros `.MSD` sueltos en la carpeta del juego |
| Amiga / ST | Todo en `PART1.DAT` |
| HTML | Arrays JavaScript en `images.js` |
| Experimentales (ADP) | Todo en un `.DAT` generado por `DMG create` |

### Formato `.GRA` del +3

Documentado en `TOOLS/plus3cache/plus3cache.php:4-19`. Se carga en `C000h + 512 + 6912`:
2 bytes de longitud, tabla de 5 bytes por imagen (`localidad 1 B`, `offset 2 B LE`,
`tamaño 2 B LE`), terminador `localidad = 255`, y después los datos.

### Formato `.NX2` / `.NXI` del Spectrum Next

- `.NX2`: 320 px de ancho, hasta 256 filas. `.NXI`: 256 px, hasta 192 filas.
- **512 bytes de paleta al principio**: 256 entradas de 2 bytes. El primer byte es `RRRGGGBB`;
  del segundo, el bit 0 es el noveno bit de azul y el bit 7 marca prioridad sobre Layer 2.
- El **índice 255 está reservado como transparente**, y el valor `0xE3` en el primer byte está
  **prohibido** por ser el color de transparencia global del Next.
- La altura se deduce de `(tamaño − 512) / ancho`.
- Comprimibles con ZX0 clásico; extensiones `.NX2.ZX0`, `.N2Z`, `.NXZ`. El cargador prueba 6
  variantes y la comprimida gana a la cruda.

---

## 6. Mapa de memoria del ZX Spectrum 128

`daad-ready/DOC/MemoryLayoutZX128K.txt`. Tres tipos de dato compiten por la RAM: **DDB**,
**XMESSAGES** e **imágenes**.

| Banco | Uso |
|---|---|
| 5 | Pantalla (7,75 KB) más el intérprete. **Inutilizable** |
| 2 | **Exclusivo del DDB**, se reserva entero aunque sobre |
| 0 | Continuación del DDB (algo menos de 14 KB); el **charset** ocupa el final; el hueco se rellena con imágenes |
| 1 | 512 B de RAMSAVE + 6912 B de buffer de pintado; el resto, imágenes |
| 3, 4, 6, 7 | XMESSAGES primero; el remanente, imágenes |

Orden de llenado: DDB en 2 → 0, charset al final del 0; XMESSAGES en 3/4/6/7; imágenes en el
remanente de 3/4/6/7 → banco 1 → banco 0.

Diagnóstico de "no cabe":

- **Banco 0 lleno** ⇒ el DDB es grande. Convertir `MES`/`MESSAGE` en `XMES`/`XMESSAGE` mueve
  texto a otros bancos.
- **Sin sitio para gráficos** ⇒ la operación inversa (poco efectiva) o simplificar imágenes: ZX0
  comprime mejor con zonas vacías y patrones repetidos.
- **Hay que borrar los `.zx0` y `.128` de `IMAGES/` para forzar la recompresión.** Es la causa
  número uno de builds que "no cambian".
- El banco 2 nunca se aprovecha para imágenes aunque el DDB sea diminuto.

---

## 7. El condacto `GFX`

`GFX p1 p2` es la puerta a las capacidades gráficas específicas de cada plataforma. La tabla
varía; esta es la común a MSX2, C64, CPC, Amiga/ST, PC/DOS y HTML:

| p2 | Efecto |
|---|---|
| 0 | Buffer → pantalla |
| 1 | Pantalla → buffer |
| 2 | Intercambiar |
| 3 / 4 | Fijar destino de gráficos |
| 5 / 6 | Limpiar |
| 7 / 8 | Fijar destino de texto (no en Amiga) |
| 9 / 10 | Escribir / leer color de paleta usando los flags `p1..p1+3` |
| 11 / 12 | Iniciar / detener *color cycling* (PC) |
| 13 / 14 | Reproducir FLI (PC) o MP4 (HTML) |
| **15** | **Split screen** — es lo que emite `XSPLITSCR` en v3 |
| 128 / 129 | Copiar la ventana actual (MSX2) |

En NextDAAD, `GFX 9`, `10` y `15` son no-op.

> **Bug en jDAAD.** Los casos 9 y 10 de `_GFX` **no tienen `break`** (`jdaad.js:3717-3718`), de
> modo que `GFX n 9` ejecuta también `DBgetPalette` y cae en el caso 13, que reproduce vídeo.
> Además los casos 5 y 6 invocan `DBClearScreen`/`DBClearBuffer` **sin paréntesis** (referencias
> a función, no llamadas), y los casos 13 y 14 hacen `_SFX;` en vez de `_SFX()`.

### Split screen

Solo CPC y C64 lo admiten desde el compilador (`drb.php:1011`). La línea de división está fija en
la scanline **96**, lo que explica que las imágenes multicolor de C64 tengan que medir 96 píxeles
de alto. Desde la versión A de DAAD Ready el split ya no es el modo por defecto en CPC ni en C64.

---

## 8. Notas para un compilador

1. Las imágenes **no forman parte del DDB**, salvo en ZXDAAD128, que las indexa dentro de su
   cabecera propia. Un compilador de DDB no necesita tocarlas, pero sí debe conocer la numeración
   para validar `PICTURE n`.
2. La única interacción real con el binario es `GFX x 15` como traducción de `XSPLITSCR` en v3.
3. `IMGLINES` afecta a la conversión, no al DDB.
4. Un compilador que quiera cubrir toda la cadena tiene que orquestar herramientas externas por
   plataforma; la secuencia exacta está en
   [11-build-plataformas.md](11-build-plataformas.md).

---

## 9. El motor vectorial de 8 bits (Aventuras AD, anterior a DAAD v2)

Todo lo anterior describe imágenes de mapa de bits. **Las aventuras españolas de 8 bits no las
usaban**: en las versiones de Spectrum y Amstrad de La Aventura Original, El Jabato o Cozumel un
dibujo no es una imagen, es **un programa** — una tira de órdenes que el intérprete ejecuta para
trazarlo en pantalla. Es otro sistema, contemporáneo del DDB v1, y no tiene nada que ver con
`SCRMAKER` ni con los `.GRA` del +3.

**El sistema está resuelto.** El juego de instrucciones (§9.1), el modelo de coordenadas (§9.2),
dónde viven los dibujos (§9.5), qué hay en la cola del bloque y con qué estado se entra a un dibujo
(§9.6) y cómo rellena el motor (§9.7). Los 708 dibujos de las cinco aventuras del Spectrum
descodifican enteros, y **15 de las 20 pantallas contrastadas con capturas del juego salen
indistinguibles del original** (§9.8).

Léase con una advertencia: casi todo lo que sigue se dedujo de un módulo y se corrigió al leer
otro. Donde §9.1 y §9.2 y las secciones de la 9.5 en adelante se contradicen, mandan las segundas,
que salen de los discos de los juegos y están contrastadas contra la pantalla.

Fuente de §9.1 a §9.3: desensamblado del intérprete `Interpreters/Spectrum/DS48IS.P3F` del DAAD de
1991, 8142 bytes, origen `0x6000` — deducido exigiendo que los saltos caigan dentro del módulo, cosa
que cumple el 92,6 % de los 530 que tiene.

> **Aviso.** El intérprete de 1991 y el de los juegos de 1989 **no miden las órdenes igual**. Todo
> lo que sigue describe el módulo de 1991; para leer los discos de los juegos hay que usar las
> longitudes de §9.5, que salen de sus propios manejadores.

### 9.1. El juego de instrucciones

El despachador está en `0x7A23`. Toma el byte de orden, se queda con los **3 bits bajos** y salta
por una tabla de 8 entradas en `0x7A3A`:

```asm
7a1d: ld e,0x03        ; longitud por defecto de una orden
7a21: add ix,de        ; ix recorre la tira de ordenes
7a23: ld a,(ix+0)
7a26: and 0x07         ; el opcode son los 3 bits bajos
7a28: sla a
7a2a: ld hl,0x7a3a     ; tabla de saltos
7a39: jp (hl)
```

| # | Dirección | Orden | Bytes | Qué hace |
|---|-----------|-------|-------|----------|
| 0 | `0x7A4A` | `PLOT` | 3 | Punto en coordenada **absoluta**, por `PLOT-SUB` de la ROM (`0x22E5`) |
| 1 | `0x7A60` | `DRAW` | 3 o **2** | Línea **relativa**, por `DRAW-LINE` de la ROM (`0x24BA`) |
| 2 | `0x7AC0` | `FILL` | 3 | Relleno con patrón; tabla de patrones en `0x7AE2` |
| 3 | `0x7B47` | `CALL` | 3 | Invoca otro dibujo, con escala y espejo |
| 4 | `0x7B9B` | `UDG` | 4 | Estampa un gráfico de 8×8 en una celda de caracteres |
| 5 | `0x7BC5` | `PAPER` | **1** | `chr$(17)`, o `BRIGHT` (`chr$(19)`) si el bit 7 |
| 6 | `0x7BDD` | `INK` | **1** | `chr$(16)`, o `FLASH` (`chr$(18)`) si el bit 7 |
| 7 | `0x7BEA` | `RET` | 1 | `pop ix`, `dec (iy+10)`; fin del dibujo |

**Las órdenes no miden todas lo mismo**, y el despachador solo pone 3 por defecto: cada manejador
carga en `E` la longitud que le toca antes de saltar a `0x7A1F`. `INK` y `PAPER` cargan 1
(`0x7BD8`) y `DRAW` carga 2 cuando lleva el bit 5 (`0x7AA6`).

> **Y no miden lo mismo en todas las versiones.** En los discos de los juegos de 1989 `CALL` ocupa
> 2 bytes, `DRAW` no tiene forma corta y `FILL` tiene tres tamaños. La tabla comparada está en
> §9.5; hay que leer cada juego con las suyas.

**Los bits altos tampoco significan lo mismo en todas.** No hay una regla global; cada orden usa
los cinco bits de arriba para lo suyo:

| Orden | bits 3-4 | bit 5 | bits 6-7 |
|---|---|---|---|
| `PLOT` | modo de trazo (`OVER`/`INVERSE`, `0x7AAB`) | — | — |
| `DRAW` | modo de trazo; **`11` es `MOVE`** | forma corta | signo de x, signo de y |
| `FILL` | modo | variante de relleno | signo de x, signo de y |
| `CALL` | escala, junto con el bit 5 | | espejo en x, espejo en y |
| `INK` / `PAPER` | valor, bits 3-6 (`rrca` ×3, `and 0x0f`) | | bit 7: `BRIGHT` / `FLASH` |

### 9.2. El modelo de coordenadas

Es lo que decide cómo se lee una tira de bytes, y no se puede adivinar del juego de instrucciones:
hay que verlo en los manejadores.

**`PLOT` es absoluto.** `0x7A50` carga `C=(ix+1)` y `B=(ix+2)` y llama al `PLOT-SUB` de la ROM. O
sea, x en el primer byte del operando e y en el segundo, tal cual.

**`DRAW` es relativo.** Llama a `DRAW-LINE` de la ROM (`0x24BA`), que traza desde `COORDS`
(`0x5C7D`) con un incremento. **Cada línea arranca donde acabó la anterior**: un dibujo es una
polilínea, no una lista de segmentos sueltos.

Y traza como traza la ROM, que no es como traza un Bresenham cualquiera: un DDA sobre el eje largo,
un punto por paso de ese eje y el corto metido en el mismo movimiento cuando el error se pasa. Las
diagonales salen diagonales. Quien reimplemente esto tiene que hacerlo igual, porque de ello depende
dónde se paran los rellenos —véase §9.8—.

> **`FILL` no mueve el lápiz.** Aquí decía que sí, y es falso — al menos en los discos de 1989,
> donde los dos manejadores se distinguen por una sola instrucción:
>
> ```asm
> ; MOVE, en 0x7C72
> cd fe 7d      call 0x7DFE        ; calcula las coordenadas nuevas en BC
> ed 43 7d 5c   ld (0x5C7D),bc     ; y las GUARDA en COORDS
>
> ; FILL, en 0x7CC5
> cd fe 7d      call 0x7DFE        ; las calcula igual...
> dd cb 00 6e   bit 5,(ix+0)       ; ...y se va a rellenar sin guardarlas
> ```
>
> O sea: el desplazamiento de un `FILL` dice **dónde rellenar**, medido desde donde esté el lápiz,
> y el lápiz **se queda donde estaba**. Una tirada de rellenos seguidos se mide toda desde el mismo
> punto en vez de ir caminando. Encadenarlos es el error que más caro ha salido de todo esto: en el
> dibujo 4 son trece seguidos, y a partir del quinto los puntos de relleno aterrizaban fuera de la
> ventana de dibujo y teñían lo que pillaran. Ver §9.8.

**Hay un `MOVE`.** Si los bits 3 y 4 están **los dos** puestos, `0x7A63` lo detecta con
`and 0x18; cp 0x18` y la orden escribe `COORDS` directamente **sin trazar** (`0x7A6F`). Así se
levanta el lápiz y se salta a otro sitio.

**El lienzo es 256 × 176**, y la y **envuelve** módulo `0xB0` = 176: al sumar, si llega a 176 resta
176 (`0x7C27-0x7C2D`); al restar, si baja de cero le suma 176 quitando `0x50` al acarreo
(`0x7C21`). Son las 22 filas de caracteres de la ventana de dibujo, y coincide con el rango que
acepta el `PLOT` de la ROM, cuya **y va de abajo arriba**.

#### Los dos tamaños de `DRAW`

`0x7C31` elige según el bit 5:

- **Larga, 3 bytes** (`0x7C50`): `E=(ix+1)`, `D=(ix+2)`, dos incrementos de 8 bits.
- **Corta, 2 bytes** (`0x7C37`): un solo byte con **un nibble por eje** — el alto para x y el bajo
  para y, así que cada incremento va de 0 a 15.

Los **signos no son complemento a dos**: salen de los bits 6 y 7, que `0x7A7E-0x7A90` convierte en
un paso de `+1` o `−1`. El operando es siempre una magnitud sin signo.

En los discos de 1989 **la forma corta no existe**: la rutina de operandos (`0x7E2C`) toma siempre
`(ix+1)` y `(ix+2)` enteros, y en efecto el bit 5 no está puesto en ninguna de las 7886 órdenes
`DRAW` del juego. Ver §9.5.

#### Escala y espejo

Es lo que hace que un dibujo no tenga vectores fijos. `CALL` (`0x7B47`) guarda el estado y lo
sustituye con lo que traiga su propio byte de orden:

- **bits 3-5, la escala** (`0x7B74`). `0x7C77` calcula `incremento × factor / 8`, así que **solo
  reduce**: 0 es tamaño natural —el transformador se salta entero— y 1 a 7 son fracciones de
  octavo.
- **bits 6-7, el espejo** (`0x7B7F`), que se aplica con un XOR sobre los bits de signo del byte de
  orden (`0x7C6D`), invirtiendo la dirección de cada eje.
- **byte 1**: el número del dibujo invocado.
- Anidamiento máximo de **10** (`0x7B4A`), con error si se pasa.

La consecuencia para quien quiera extraer: **hay que llevar la escala y el espejo como estado**.
Los mismos bytes de un sub-dibujo dan vectores distintos según con qué se le llamó.

#### La orden 4 no es un `AT`, es un `UDG`

El manejador de `0x7B9B` hace más que posicionar. Emite `chr$(22)` con la columna del byte 2 y la
fila del byte 3, y antes toma el **byte 1**, lo multiplica por 8 (`add hl,hl` tres veces), le suma
`(0xFFF7)` y guarda el resultado en `0x5C7B` — que es la variable del sistema **`UDG`**. O sea que
**estampa un gráfico de 8×8 definido por el usuario** en una celda de caracteres.

Es la forma barata de repetir texturas: ladrillo, follaje, sombreado. Y `(0xFFF7)` es la base de la
tabla de gráficos, otra casilla que rellena el cargador igual que `(0xFFF1)`.

#### No hay círculos

El motor **nunca llama** al `CIRCLE` de la ROM (`0x2320`) ni al `DRAW` con arco (`0x2382`);
comprobado buscando los `call` en todo el binario. Solo usa `PLOT-SUB` y `DRAW-LINE`. Las
primitivas son **el punto y la recta**, y todo lo curvo va aproximado con segmentos.

#### Cómo queda un listado

Llevando la cuenta de `COORDS` se pueden dar los extremos en absoluto, que es lo legible:

```text
INK 6
PLOT 100,50              ; absoluta
LINE 120,65              ; +20,+15 desde donde estaba
LINE 120,90
MOVE 40,40               ; sin trazar
LINE 60,40
CALL 12 escala=4 espejo=x
```

### 9.3. Cómo se localiza un dibujo

La rutina de entrada (`0x79B0`) convierte un número de dibujo en un puntero:

```asm
79b4: ld hl,(0xfff1)   ; base de la tabla de dibujos
79b8: add hl,hl        ; numero x 2  -> entradas de 16 bits
79b9: add hl,de        ; + base
79ba: ld e,(hl) / inc hl / ld d,(hl)
79bd: push de / pop ix ; ix = la tira de ordenes del dibujo
79ce: call 0x7a23      ; a interpretarla
```

Es decir: **el dibujo N está en `[(0xFFF1) + N×2]`**, una tabla plana de punteros de 16 bits. La
orden `CALL` usa **esa misma tabla**, así que cualquier dibujo puede invocar a otro por número —
que es lo que hace que quepan tantos: los elementos repetidos se trazan una vez y se llaman desde
donde hagan falta.

#### Hasta el extremo de que una entrada sea solo un alias

Con esa mecánica, una localidad que comparte ilustración con otra no repite nada: su entrada de la
tabla apunta a tres bytes.

```text
IMAGE 014        IMAGE 015        IMAGE 017        IMAGE 018        IMAGE 019
CALL 61,0,0      CALL 61,0,0      CALL 61,0,0      CALL 61,0,0      CALL 61,0,0
RET              RET              RET              RET              RET
```

En la parte 2 de La Aventura Original **13 de las 108 entradas son un `CALL n; RET`** y nada más:
seis apuntan al dibujo 63, cinco al 61. Con 40 KB para el juego entero, repetir un dibujo de mil
bytes seis veces no era una opción.

La reutilización también es más fina cuando conviene. En la parte 1, las localidades 13, 14 y 15
comparten el mismo bosque y cada una le añade su detalle:

```text
013: CALL 42                             ; el bosque, a secas
014: CALL 42 / FILL -136,-53,132
015: CALL 42 / PLOT 66,94 / FILL -56,-28,132
```

Ahí el 14 rellena en (9,67) y el 15 en (10,66) — a un píxel, dentro de la misma región cerrada. Las
dos pantallas salen idénticas aunque su código no lo sea, y así se ven también en el Spectrum.

De las 41 imágenes que la parte 1 llega a dibujar, 40 son distintas; de las 69 de la parte 2, 55. Un
extractor que encuentre imágenes repetidas no tiene por qué estar fallando.

### 9.4. El motor está también en los discos del juego

No es solo material del DAAD de 1991. En el disco de La Aventura Original de Spectrum la firma del
despachador aparece **dos veces, una por parte**, en los offsets `0x482E` y `0x1042D` de la carga
útil. No es casualidad de bytes: dos firmas independientes —el despachador y el `PLOT` por ROM—
guardan entre sí exactamente la misma distancia que en el original, y los 8 manejadores de la
tabla de saltos caen los 8 dentro del fichero. En las dos partes el motor vive en las mismas
direcciones absolutas, que están tabuladas más abajo.

En la carga útil del Amstrad **la firma no aparece**: o su motor está ensamblado de otra forma, o
llega comprimido.

#### Dónde está exactamente, y la correspondencia que faltaba

El despachador del disco hace `ld hl,0x7C48`, y con la disposición del módulo de 1991 —la tabla de
saltos 0x17 bytes después del despachador— eso lo sitúa en `0x7C31`. De ahí sale la herramienta que
faltaba para leer cualquier cosa de esta zona:

> - Parte 1: **dirección = offset de la carga útil + `0x3403`**
> - Parte 2: **dirección = offset de la carga útil − `0x87FC`** (o sea `0x3403 − 0xBBFF`)

La correspondencia se comprueba sola: los 8 manejadores de la tabla de `0x7C48` caen contiguos al
despachador y su código coincide instrucción a instrucción con §9.1 y §9.2 —`PLOT` hace
`ld b,(ix+2) / ld c,(ix+1) / call 0x22E5`, y `DRAW` hace `and 0x18 / cp 0x18` y luego
`ld (0x5C7D),bc`—.

| Rutina | Dirección | Rutina | Dirección |
|---|---|---|---|
| Punto de entrada (§9.3) | `0x7BF8` | `CALL` | `0x7D41` |
| Despachador | `0x7C31` | `UDG` | `0x7D95` |
| Tabla de saltos | `0x7C48` | `PAPER` | `0x7DBF` |
| `PLOT` | `0x7C58` | `INK` | `0x7DD5` |
| `DRAW` | `0x7C69` | `RET` | `0x7DE2` |
| `FILL` | `0x7CBD` | Operandos, escala y espejo | `0x7E2C` |

### 9.5. Dónde están los dibujos

**Resuelto.** El bloque está localizado en el disco de Spectrum de La Aventura Original, y los 157
dibujos que contiene —49 en la parte 1, 108 en la parte 2— descodifican enteros: los 157 terminan en
`RET` justo en el byte donde empieza el siguiente, ninguna `y` de `PLOT` pasa de 175 y ninguna
`CALL` apunta fuera de la tabla.

Lo que lo desbloqueó no fue rastrear el disco, sino **mirar la cinta**. Y de paso hubo que corregir
las longitudes de las órdenes, que en los juegos de 1989 no son las del módulo de 1991.

#### La idea que sí era buena

El diagnóstico de partida —que hacía falta entender el cargador para saber qué se escribe en
`(0xFFF1)`— era erróneo, y por la razón que ya se apuntaba: **el dato viaja con los datos**. Ahora
está confirmado por ausencia. En los 195.584 bytes de la carga útil no hay ni un `ld (0xFFF1),hl` ni
un `ld (0xFFF1),de`; solo lecturas, en `0x7BF8` (la rutina de entrada de §9.3) y en `0x7D87` (dentro
del manejador de `CALL`). Nadie escribe ese puntero en tiempo de ejecución: llega ya puesto dentro
del bloque que se carga.

En el **DAAD de 1991** eso se ve en su propio fichero de gráficos, el `.SDG` del Spectrum, que es un
fichero PLUS3DOS con sus 128 bytes de cabecera. La de `PART1.SDG` dice tipo 3 (CODE), 2089 bytes,
carga en `0xF7D7`, y `0xF7D7 + 2089 = 0x10000`: el bloque llega justo al techo de la memoria, de
modo que `0xFFF1` y `0xFFF7` caen en su cola, en los offsets `0x081A` y `0x0820` del propio fichero.
En el SDG en blanco la tabla queda en `0xF7D8` y sus entradas apuntan a un byte `0x07` —un `RET`,
dibujos que no hacen nada—, coherente con estar vacío y por eso mismo inútil para validar contenido.

Ese reparto es del sistema de 1991, y **suponer que valía también para los juegos de 1989 fue el
error de partida**: allí no hay fichero de gráficos aparte.

#### La respuesta estaba en la cinta

Buscar la estructura a ciegas por el disco era el camino largo. Las versiones de cinta del juego
están en el mismo repositorio, y **una cabecera de cinta del Spectrum declara nombre, longitud y
dirección de carga**, sin que haya que adivinar nada. Las dos cintas dicen lo mismo:

| Bloque | Tipo | Longitud | Carga | Fin |
|---|---|---|---|---|
| `ao1` / `ao2` | PROGRAM | 284 | — | — |
| `AO$` | CODE | 6924 (`0x1B0C`) | `0x7530` | `0x903C` |
| `codigo` | CODE | 40444 (`0x9DFC`) | `0x6204` | **`0x10000`** |

`codigo` es un bloque único de 40.444 bytes que termina exactamente en el techo de la memoria. No
hay un fichero de gráficos aparte: **el intérprete, sus datos y los dibujos van todos dentro**, y
`0xFFF1` y `0xFFF7` caen en su cola. Con la correspondencia de §9.4 el bloque se localiza en la
carga útil del disco sin más:

| | Parte 1 | Parte 2 |
|---|---|---|
| Bloque `codigo` en la carga útil | `0x02E01`–`0x0CBFD` | `0x0EA00`–`0x187FC` |
| `(0xFFF1)`, base de la tabla de dibujos | `0xF688` | `0xF4EB` |
| `(0xFFF7)`, base de la tabla de UDG | `0xF7E4` | `0xF7E4` |
| Tabla de dibujos en la carga útil | `0x0C285` | `0x17CE7` |
| Datos de los dibujos | `0x081F0`–`0x0C285` | `0x14C60`–`0x17CE7` |
| Número de dibujos | **49** | **108** |

En el techo de la memoria hay un bloquecito de cuatro punteros —`0xFFF1`, `0xFFF3`, `0xFFF5` y
`0xFFF7`—. El motor solo usa el primero y el último, pero el segundo marca dónde acaba la tabla de
dibujos, que es de donde sale cuántos hay.

La tabla es plana, de 16 bits por entrada, y sus punteros salen **distintos y estrictamente
crecientes**. Las dos últimas entradas no son dibujos:

- la penúltima vale exactamente **la dirección de la propia tabla**, y sirve para acotar el último
  dibujo — el idioma habitual de N+1 punteros;
- la última es un **`0xFFFF` de centinela**.

Así que el número de dibujos es `((0xFFF3) − (0xFFF1)) / 2 − 2`: 51 entradas dan 49 dibujos en la
parte 1, y 110 dan 108 en la parte 2. Los dibujos se guardan seguidos, en orden, y acaban justo
donde empieza su índice.

Las 49 de la parte 1 encajan con las 49 localidades del juego, y entre ellas hay tanto
ilustraciones completas como piezas que solo existen para que otras las llamen: en la parte 1 se
invocan por `CALL` los dibujos 2, 26 y 30 a 45.

#### Las longitudes de orden son otras en 1989

Este es el motivo real de que el encadenado no saliera antes. Leídas en los manejadores del propio
disco, **tres órdenes no miden lo que en el módulo de 1991**:

| Orden | 1991 (§9.1) | Discos de 1989 | De dónde sale |
|---|---|---|---|
| `PLOT` | 3 | 3 | `0x7CA5` → `0x7C23` |
| `DRAW` | 3, o **2** con el bit 5 | **3 siempre** | `0x7E2C` toma `(ix+1)` y `(ix+2)` enteros; no hay forma corta |
| `FILL` | 3 | **3, 4 o 5** | `0x7CD7`, `0x7CFE`, y `0x7D33` para el byte `0x12` |
| `CALL` | 3 | **2** | `0x7DF9`: al volver, el `RET` hace `ld e,2` |
| `UDG` | 4 | 4 | `0x7DB9` |
| `INK` / `PAPER` | 1 | 1 | `0x7DBF` y `0x7DD5` → `0x7C28` |
| `RET` | 1 | 1 | `0x7DE2` |

Las dos que importan:

- **`CALL` ocupa 2 bytes**, no 3. El manejador (`0x7D8F`) salta al despachador sin tocar `IX`; es
  el `RET` del sub-dibujo el que restaura `IX` y le suma 2. Con 3 se desincronizaba la tira entera
  justo después de la primera llamada, que es lo que hacía que los dibujos «terminaran» en el
  primer byte con los 3 bits bajos a 7.
- **`FILL` tiene tres tamaños.** Son 3 bytes normalmente; 4 si lleva el bit 5, porque entonces el
  patrón se toma de la tabla de UDG con el byte 3 como índice (`0x7CE4`); y 5 en el caso especial
  del byte de orden `0x12` exacto, que no es un relleno de trama sino un **rectángulo de
  atributos**: calcula una dirección en `0x5800` y pinta desde ahí.

Los cuatro operandos del `0x12` son, en el orden en que están, **filas** (`0x7D17`), **columnas**
(`0x7D1A`), **columna inicial** (`0x7D0A`) y **fila inicial** (`0x7D06`); los dos contadores valen
uno menos de lo que se pinta. Lo confirma un grupo de siete dibujos de la parte 2 que no son
dibujos sino **rutinas de limpieza**, idénticas salvo el color:

```text
MOVE 0,64
INK 0
PAPER n            ; una por cada color: 1, 2, 3, 4, 5, 6 y 7
ATTR 13,31,0,0     ; 14 filas x 32 columnas desde el origen
RET
```

Y de paso dan la medida a la que trabajan casi todas las escenas: 14 filas de caracteres, o sea
**112 píxeles**, que es justo lo que queda por encima de la línea `y = 64` con la que abren. Es una
costumbre de los dibujos, no un límite que nadie imponga —véase §9.6—.

Las tres longitudes de `FILL` aparecen en los datos reales (308, 148 y 63 veces en la parte 1), así
que ninguna es especulativa: si una fuera falsa, la tira se desincronizaría y el dibujo no
terminaría donde termina.

#### La comprobación

Descodificados los 157 dibujos de las dos partes con esas longitudes:

- **157 de 157** terminan en `RET` exactamente en el byte donde empieza el siguiente. Los dibujos
  cubren su región sin huecos ni solapes.
- La `y` máxima de un `PLOT` es **175** en las dos partes — el borde justo del lienzo de 176 de
  §9.2, tocado pero no rebasado ni una vez en 513 órdenes `PLOT`.
- Ninguna `CALL` referencia un dibujo fuera de la tabla, y ningún `UDG` cae fuera de la rejilla de
  32 × 22 caracteres.
- El bit 5 de `DRAW` no está puesto **en ninguna** de las 7886 órdenes `DRAW` de las dos partes,
  que es lo que cabe esperar si en esta versión no existe la forma corta.

Y las coordenadas son las de un dibujo, no las de un montón de bytes. El dibujo 30 de la parte 1
—el que invocan varias localidades— empieza así:

```text
MOVE 0,64                ; sube el lápiz al borde superior de la ventana
LINE 255,64              ; +255,+0: la línea del horizonte, de lado a lado
PLOT 208,136
LINE 178,148             ; y de aquí, una silueta quebrada de sierra
LINE 178,142
LINE 175,149
LINE 158,164
...                      ; 221 órdenes en total
```

Y una localidad que lo reutiliza, el dibujo 3, es exactamente la composición que anticipaba §9.3:

```text
CALL 30                  ; el fondo
PLOT 55,70
CALL 35                  ; una pieza suelta
LINE 59,64
MOVE 78,64
LINE 77,65
...                      ; 256 órdenes en total
```

#### Y se ha comprobado mirándolos

Lo anterior demuestra que las tiras se leen sin desincronizarse, que es otra cosa que dibujar bien.
Así que se han trazado los 708 dibujos de las cinco aventuras sobre un lienzo de Spectrum,
emulando los atributos por celda de 8 × 8 —imprescindible, porque muchas escenas trazan con
`INK 0` sobre un `PAPER` de color y sin papel no se ve nada—. Ninguno da error y ninguno sale
ilegible.

Salen **ilustraciones**: una casa con su puerta y su tejado bajo un cielo cian, bosques, cuevas con
un portón, el logotipo «AD» de Aventuras AD, y un interior en perspectiva con el suelo entablado
cuyas líneas de fuga convergen como es debido — que es la prueba más dura de todas, porque un solo
byte mal leído la destrozaría.

De los 49 de la parte 1, los **únicos** que salen en negro son los 8 que solo contienen `RET`. El
resto se reparte entre 26 escenas completas y 15 piezas que existen para ser invocadas con `CALL`
desde ellas y que, trazadas sueltas, no tienen por qué componer nada.

Aquello se trazó sin rellenar y sin estampar los `UDG`, que era todo lo que se sabía hacer
entonces. Las dos cosas están ahora resueltas —§9.7 para el relleno, §9.6 para los caracteres— y
las ilustraciones salen completas.

#### El candidato anterior era ruido

Conviene dejar constancia de por qué, porque el error estaba en el criterio de búsqueda, no en los
datos. El candidato era «tabla en `0xFF05`, 45 dibujos, en los offsets `0x04554` y `0x10153`».

1. **Los dos punteros eran lecturas accidentales.** Las palabras que hacían de `(0xFFF1)` y
   `(0xFFF7)` eran los pares `05 ff` y `5f ff` dentro de una tira de relleno de `0xFF`. Los 90
   bytes de la supuesta tabla son `0xFF` macizo y luego un registro de 8 bytes repetido; las 16
   entradas que caían «en rango» valían **todas** `0xFFFF`.
2. **Las «45 entradas» no contaban nada.** Son `(0xFF5F − 0xFF05) / 2`, la distancia entre las dos
   palabras accidentales. El número lo fabricaba el propio criterio.
3. **El filtro no discriminaba.** Solo exigía un byte seguido de `0xFF`, dos veces, a 6 bytes de
   distancia y en orden ascendente. El 3,1 % de la carga útil son bytes seguidos de `0xFF`.
4. **La coincidencia de `0xBBFF` era circular.** La parte 2 es una copia casi literal de la parte 1
   desplazada `0xBBFF`, así que *cualquier* patrón de la parte 1 reaparece a `+0xBBFF`. Que el
   candidato lo hiciera no aportaba información.
5. **Y la dirección de carga lo desmentía.** Con la correspondencia de §9.4, el candidato exigía
   que el offset `0x04D6E` estuviera en `0xFFF1`, cuando está en `0x8171`: caía dentro del módulo
   del intérprete, 730 bytes antes del despachador.

Descodificado a mano, el primer byte del bloque candidato es `0x4F`, y `0x4F & 7 = 7`, o sea `RET`;
el destino de su primera entrada es `0xFF`, otro `RET`. La prueba de las coordenadas no llegaba
siquiera a tener un número que mirar.

La lección para la siguiente plataforma: un criterio de búsqueda que solo mire la *forma* de unos
punteros no vale. Lo que decide es exigir que **las entradas sean distintas**, que cada dibujo
**termine en `RET` justo donde empieza el siguiente**, y que las coordenadas caigan en el lienzo.
Y antes de nada, mirar si hay una versión en cinta que declare la dirección de carga.

#### Cada juego trae su propio motor

Al llevar esto al descompilador salió lo que un solo juego no podía enseñar: **las longitudes de
orden no son una propiedad del año, sino de cada compilación**. El motor está en direcciones
distintas en cada juego, y con él cambian las longitudes:

| Juego | Tabla de saltos | Manejadores | `CALL` | `DRAW` | `FILL` | Dibujos |
|---|---|---|---|---|---|---|
| La Aventura Original | `0x7C48` | `0x7C58`… | 2 | 3 | 3, 4 o 5 | 49 + 108 |
| El Jabato | `0x7C5E` | `0x7C6E`… | 2 | 3 | 3, 4 o 5 | 59 + 62 |
| Cozumel | `0x7BA0` | `0x7BB0`… | 2 | **3, o 2 con el bit 5** | 3, 4 o 5 | 30 + 33 |

**Cozumel sí tiene la forma corta de `DRAW`** —un nibble por eje— que La Aventura Original y El
Jabato no usan nunca. Es posterior, y en eso se acerca al módulo de 1991.

Por eso el descompilador **no fija las longitudes: las deduce**. Prueba los dialectos posibles y se
queda con el que consigue que *todos* los dibujos de la tabla terminen en `RET` justo donde empieza
el siguiente. Una longitud equivocada por un byte desincroniza la tira y se delata ahí mismo, así
que el criterio que validó el bloque sirve también para elegir cómo leerlo. Donde varios dialectos
valen, es que el juego no usa los bits en que difieren, y el resultado es el mismo.

Quedaba una duda con la orden `0x12`. En La Aventura Original su manejador calcula una dirección en
`0x5800` con la fila y la columna de los bytes 3 y 4 —un rectángulo de atributos—, y aquí se dijo
que Cozumel ponía ahí valores que no podían significar eso. **Se leen igual, y está comprobado
contra el juego.** Las escenas de Cozumel abren con

```asm
ATTR 12,31,0,1     ; 13 filas x 32 columnas, desde la columna 0 y la fila 1
```

que son exactamente las filas de píxel 8 a 111, y es justo donde la captura pone el marco: Cozumel
reserva la fila 0 para su rótulo —«Plaza», «Náufrago», y el contador de turnos a la derecha— y baja
el dibujo una fila de caracteres, cosa que La Aventura Original no hace. Lo que parecía un operando
imposible era una ventana colocada de otra manera.

#### Buscar el motor era el método equivocado

Anclar en la firma del despachador funcionaba con los tres juegos de arriba porque en ellos el
motor y los dibujos viajan en el mismo bloque. **Los Templos Sagrados y Chichén Itzá no lo hacen**:
su cargador es propio y mete el intérprete y los gráficos en bloques distintos y sin cabecera, así
que junto a los dibujos no hay ningún despachador del que sacar la dirección de carga.

La salida está en lo que ya decía §9.5 y no se estaba aprovechando: **el bloque de gráficos siempre
acaba en `0x10000`**. Eso basta para localizarlo sin saber nada del cargador, porque fija la
dirección de todos sus bytes. Se prueba cada offset como si fuera ese final, y la propia
disposición acepta o rechaza: la tabla tiene que ser creciente, acabar apuntándose a sí misma y
cerrar con `0xFFFF`, y luego **todos** sus dibujos tienen que terminar en `RET` justo donde empieza
el siguiente.

Suena indiscriminado y no lo es. Sobre las 6 ediciones de Spectrum el barrido deja **uno o dos
candidatos por medio, y todos son reales**: ni un falso positivo. Es el contraste exacto con el
criterio viejo, que sobre el mismo disco dejaba cientos. Y sale gratis: 27 ms para los 195 KB del
disco de La Aventura Original.

Con eso el descompilador saca los dibujos de los 5 juegos, tanto de disco como de cinta:

| Juego | Dibujos | De dónde |
|---|---|---|
| La Aventura Original | 49 + 108 | `.dsk` y `.tzx` |
| El Jabato | 59 + 62 | `.dsk` |
| Cozumel | 30 + 33 | `.dsk` |
| Los Templos Sagrados | 60 + 58 | `.tzx` |
| Chichén Itzá | 21 + 38 | `.tzx` |

Los dos últimos **solo salen de la cinta**. En el caso de Los Templos Sagrados no es por los
gráficos: su `.dsk` falla antes, al leer `PART1.DDB`, y eso es otro asunto.

Una nota sobre el intérprete: en el disco de Los Templos Sagrados la firma del despachador aparece
con su tabla de saltos en `0x7A3A`, que es la del módulo de 1991 (`DS48IS.P3F`, despachador en
`0x7A23`), y uno de los bloques de su cinta mide 8142 bytes, exactamente lo que ese módulo. Pero
sus dibujos **no** se leen con las longitudes que §9.1 atribuye a 1991: el dialecto que los
descodifica enteros es el de Cozumel, con `CALL` de 2 bytes. O el desensamblado de §9.1 se equivoca
en ese punto, o el motor de dibujo no es el mismo que el intérprete. Queda abierto.

### 9.6. La cola del bloque: cuatro punteros, cuatro tablas

En la cola del bloque hay cuatro punteros seguidos, y durante un tiempo solo se supo para qué
servían dos. Están los cuatro resueltos, y **cada uno es la base de una tabla que acaba justo donde
empieza la siguiente**:

| Puntero | Apunta a | Qué hay | Acaba en |
|---|---|---|---|
| `(0xFFF1)` | tabla de dibujos | punteros de 16 bits, uno por dibujo | `(0xFFF3)` |
| `(0xFFF3)` | tabla de propiedades | **5 bytes por dibujo** | `(0xFFF5)` |
| `(0xFFF5)` | — | cierra la anterior | — |
| `(0xFFF7)` | juego de caracteres | 8 bytes por carácter | la cola de punteros |

Que las dos primeras encajen al byte —`5 × número de dibujos` cabe exactamente entre `(0xFFF3)` y
`(0xFFF5)` en **los siete bloques** de las cinco aventuras— es a la vez la confirmación de la
lectura y un filtro excelente para localizar el bloque.

En La Aventura Original, parte 1: `0xF688` la de dibujos, `0xF6EE` la de propiedades (49 × 5 = 245
bytes hasta `0xF7E3`), y `0xF7E4` el juego de caracteres.

#### Con qué estado se entra a un dibujo

No hay que adivinarlo: **está tabulado**. El intérprete localiza el registro del dibujo en `0x7C18`
—cinco bytes indexados por su número— y lo aplica en `0x7BBC`:

```asm
7C18: 2a f3 ff     ld hl,(0xFFF3)   ; base de la tabla de propiedades
7C1B: 16 00        ld d,0
7C1D: 19 19 19 19 19   add hl,de x5 ; + 5 x numero de dibujo

7BD9: e6 3f        and 0x3f         ; byte 0, sin bright ni flash
7BE4: 22 8d 5c     ld (0x5C8D),hl   ; -> ATTR-P
7BE7: 22 8f 5c     ld (0x5C8F),hl   ; -> ATTR-T
7BEA: 6c           ld l,h
7BEB: 22 7d 5c     ld (0x5C7D),hl   ; -> COORDS = 0,0
```

| Byte | Destino | Qué es |
|---|---|---|
| 0 | `ATTR-P` y `ATTR-T` | atributo: tinta en los bits 0-2, papel en los 3-5 |
| 1 | `0x8163` | fila de la ventana |
| 2 | `0x8164` | columna de la ventana |
| 3 (menos 1) | `0x8162` | filas de alto |
| 4 | `0x8161` | columnas de ancho |

Así que un dibujo empieza con **el lápiz en el origen** y con la tinta y el papel que le tocan. En
La Aventura Original solo hay dos combinaciones: blanco sobre negro en 40 dibujos y blanco sobre
azul en 9. El `and 0x3f` es lo que deja fuera el brillo.

Y cada dibujo declara su propia ventana. No todos la misma: en La Aventura Original salen tres
alturas.

| Ventana | Píxeles | Dibujos |
|---|---|---|
| 12 filas | 96 | 46 |
| 14 filas | 112 | 61 |
| 22 filas | 176 | 3 |

**Pero la ventana no recorta el dibujo.** Es una tentación creerlo —las localidades caben justas en
sus 14 filas y por debajo va el texto de la aventura— y estuvo un tiempo escrito aquí como si lo
fuera. Lo desmiente el dibujo 44 de la parte 1, la pantalla de créditos: su registro declara 14
filas y el dibujo pinta las 22. Su marco va de la fila 24 a la 151 de pantalla, que es exactamente
lo que dice su propia orden `ATTR 15,25,3,3` —filas 3 a 18, columnas 3 a 28—, y la captura del juego
lo confirma: hay tinta desde la fila 0 hasta la 175.

La razón es que el motor no recorta nada. Cada coordenada acaba en `PLOT-SUB` de la ROM, que solo
limita a la pantalla; el recorte de 24 filas y 42 columnas de `0x76F7` está en la rutina de
impresión, no en la de dibujo. Lo que la ventana gobierna es **dónde escribe el intérprete después**,
y un dibujo es muy libre de pintar fuera. Por eso el descompilador la anota en la cabecera del
`.plot` y traza el lienzo entero.

Los tres de 22 filas ocupan la pantalla entera y el juego les escribe el texto encima: el dibujo 43
de la parte 2 —el enano lanzando el hacha— tiene bajo el texto medio hacha que en pantalla no llega
a verse.

#### Y el fondo es ese mismo atributo

Una celda que el dibujo no llega a tocar no se queda en blanco ni en negro: muestra el atributo del
registro, porque el intérprete ya lo ha escrito en `ATTR-P` y `ATTR-T` antes de empezar y **el motor
no limpia la pantalla**.

Parece un detalle y no lo es. Tomar por fondo, digamos, el primer `PAPER` que emite el dibujo
—tentador cuando aún no se sabe leer el registro— sale bien en las escenas que cubren todo el
lienzo y falla justo en las que dejan margen: el dibujo 43 de la parte 2 pierde así su marco entero,
1942 píxeles de los 27 940 que se comparan.

#### La pantalla tiene 42 columnas, no 32

El recorte de la ventana (`0x76F7`) da los límites, y el segundo sorprende:

```asm
76FD: fe 18        cp 0x18          ; 24 filas
770F: fe 2b        cp 0x2b          ; 43 -> o sea 42 columnas
```

42 columnas en 256 píxeles son **6 píxeles por carácter**, y no es deducción: la conversión de
columna a byte de pantalla está en `0x7B1A`.

```asm
7B1A: 5f           ld e,a           ; e = columna
7B1B: cb 3b cb 3b  srl e / srl e    ; e = columna / 4
7B21: 19 19 19     add hl,de x3     ; 3 bytes por cada 4 columnas
```

Cuatro caracteres ocupan tres bytes, o sea 24 píxeles. Por eso la rutina de impresión escribe **dos
bytes de pantalla por fila** con máscaras (`0x798F`): un carácter de 6 píxeles casi nunca cae
alineado.

Y confirma por otra vía algo que ya estaba anotado en el descompilador: los juegos de caracteres de
estas aventuras dibujan **6 píxeles dentro de los 8** de la celda, cosa que allí se dedujo del
nombre de uno de ellos, `AO6X8.BIN` (ver `cmd/qundaad/decompiler/assets.go`). Dos caminos
independientes, la misma anchura.

> **Y aun así, el `UDG` va a 8 píxeles.** Parece que debería usar esta rutina —su manejador emite
> `chr$(22)` y el carácter por `RST 0x10`— y de ahí se dedujo un tiempo que se estampaba cada 6.
> Es falso, y lo zanjó una captura: el dibujo 40 de la parte 2 pone una letra en la columna 8 y
> aparece en `x = 64`, no en `x = 48`. `RST 0x10` va a la impresión ordinaria de 32 columnas; la
> de 42 es la del intérprete de texto, y el motor de dibujo no pasa por ella.
>
> Así que la anchura de 6 píxeles vale para el texto de la aventura y no para lo que dibuja el
> motor. Conviene recordarlo antes de volver a caer.

#### La tabla de `(0xFFF7)` no es «de UDG»: es el juego de caracteres

`0x797A` hace `ld de,(0xFFF7)` para imprimir **cualquier** carácter. Dibujada como atlas, la tabla
se lee sola:

| Rango | Qué hay |
|---|---|
| 16-31 | acentos y signos españoles: `ª ¡ ¿ « » á é í ó ú ñ Ñ ç Ç ü Ü` |
| 32-127 | ASCII |
| **128-159** | **las tramas de relleno** — dameros, rayados, diagonales |
| 160-256 | un segundo juego, en negrita, con iconos |

Encaja con el uso: los `FILL` con patrón tiran de 128-153 y las órdenes `UDG` de 196-218.

### 9.7. El relleno, orden por orden

La rutina está en `0x7E6B` y no se parece a un relleno moderno en nada. Tres cosas la definen:

**Son dos pasadas, no una.** Baja desde la semilla hasta agotarse y luego repite la pasada entera
hacia arriba, empezando una línea más allá:

```asm
7F9E: ed 4b ee 7f  ld bc,(0x7FEE)   ; las coordenadas de partida
7FA2: cb 7a        bit 7,d
7FA4: 20 09        jr nz,0x7FAF     ; la segunda ya se hizo -> salir
7FA6: cb fa        set 7,d
7FA8: 04           inc b            ; y+1
7FAC: da bb 7e     jp c,0x7EBB      ; y otra vez, ahora hacia arriba
```

Cada pasada avanza en un solo sentido: `bit 7,d` elige entre `inc b` y `dec b`, y `d` nace a cero,
así que la primera baja. El array de 176 bytes —uno por scanline, reservado en `0x7EA5` con 88
`push`— **no se reinicia entre las dos**, y es lo que impide que la segunda rehaga el trabajo de la
primera.

**Se rinde.** `0x7EC3` carga 30 en `A'`, y cada rama que se apila lo decrementa:

```asm
7F55: 3d           dec a
7F56: 28 46        jr z,0x7F9E      ; agotadas -> abandona la pasada a medias
7F59: c5           push bc          ; solo entonces se apila
```

No es un error: el relleno **deja la región a medio pintar** y sigue. Es lo que produce los
sombreados que se quedan cortos en los originales.

**Colorea aunque no pinte.** `0x7F29` llama a la rutina de atributos en cada píxel que recorre,
encienda o no el patrón; y `or (hl)` en `0x7F27` solo enciende, nunca apaga. La fila del patrón
sale de `y & 7` con la `y` del motor, por código automodificable:

```asm
7EE8: e6 07        and 7            ; y mod 8
7EEA: 32 ef 7e     ld (0x7EEF),a    ; escribe el desplazamiento de...
7EED: dd 56 00     ld d,(ix+NN)     ; ...esta misma instruccion
```

#### El modo del lápiz, en `0x7CA8`

Los bits 3 y 4 de un `PLOT` o un `DRAW` eligen cómo se pone el píxel. La rutina los dobla en
`P-FLAG` con una cuenta que no se adivina mirando:

```asm
7CA8: 0f 0f 0f     rrca x3          ; los bits 3-4 al fondo
7CAB: e6 03        and 3
7CAE: c6 02        add a,2          ; (modo + 2) and 5
7CB0: e6 05        and 5            ; bit 0 = OVER, bit 2 = INVERSE
7CB3: 3a 91 5c     ld a,(0x5C91)    ; P-FLAG
7CB6: e6 fa        and 0xfa         ; conservando el resto
```

| Modo | P-FLAG | Qué hace |
|---|---|---|
| 0 | 0 | pone el píxel |
| 1 | 1 | `OVER`: lo invierte |
| 2 | 4 | `INVERSE`: lo borra |
| 3 | 5 | ambos; la `PLOT-SUB` de la ROM mira `OVER` primero, así que invierte |

En un `DRAW` el modo 3 no llega aquí: se intercepta antes como `MOVE`.

El relleno no pasa por esta rutina —su manejador no la llama— y escribe con `or (hl)`, así que
siempre pone píxeles.

> Se usa poco y se nota menos: 15 órdenes en toda la parte 1, casi todas un `PLOT` suelto sobre
> pantalla en blanco, donde invertir y poner el píxel dan lo mismo. Implementarlo no cambió un solo
> píxel de las 38 pantallas contrastadas. Está aquí porque el motor lo hace, no porque se vea.

#### El `0x12` es un rectángulo de atributos

De los tres tamaños de `FILL`, el byte de orden `0x12` exacto no rellena: pinta directamente en
`0x5800`. Sus cuatro operandos son, en el orden en que están, **filas** (`0x7D17`), **columnas**
(`0x7D1A`), **columna inicial** (`0x7D0A`) y **fila inicial** (`0x7D06`), y los dos contadores
valen uno menos de lo que se pinta.

Lo confirma un grupo de siete dibujos de la parte 2 que no son dibujos sino **rutinas de limpieza**,
idénticas salvo el color:

```text
MOVE 0,64
INK 0
PAPER n            ; una por cada color: 1, 2, 3, 4, 5, 6 y 7
ATTR 13,31,0,0     ; 14 filas x 32 columnas desde el origen
RET
```

### 9.8. Contrastado con capturas del juego

Todo lo anterior se ha comprobado contra **capturas reales de 38 pantallas** del Spectrum, 20 de la
parte 1 y 18 de la parte 2, y es lo que ha permitido cerrar el asunto. Antes de tenerlas se
trabajaba a ciegas: un dibujo puede parecer razonable y estar mal.

Lo que destaparon, por orden de daño:

- **El `FILL` no mueve el lápiz** (§9.2). Es el fallo que lo torcía todo, y solo se ve marcando el
  punto de partida de cada relleno sobre el trazado: caía en cielo abierto en vez de dentro de la
  nube que tocaba rellenar. En el dibujo 4 hay trece rellenos seguidos, y a partir del quinto los
  puntos aterrizaban fuera de la ventana.
- **Un relleno con la semilla en la fila de arriba se perdía entero.** La segunda pasada empieza en
  `y + 1`, que ahí se sale del lienzo; el motor la salta y conserva lo que la primera ya pintó,
  porque pinta sobre la marcha. Un emulador que acumule la región para pintarla al final tiene que
  salir de ese bucle con cuidado de no tirarla.
- **El `UDG` se estampa cada 8 píxeles**, no cada 6, por más que el intérprete tenga una rutina de
  texto de 42 columnas (§9.6).
- **Ningún relleno se desborda.** Con el lápiz quieto, todas las regiones cierran. Cualquier
  «salvaguarda» contra rellenos que se escapan estaba tapando aquel fallo, y de paso descartaba
  regiones buenas: la montaña del dibujo 3 ocupa 15 022 píxeles.
- **Una celda que el dibujo no toca muestra el atributo de su registro**, no el primer `PAPER` que
  el dibujo emita (§9.6). Solo se nota en las escenas que dejan margen, y ahí se nota entero.
- **El tope de 30 ramas no llega a alcanzarse** en ninguna de estas pantallas. Está en el motor y se
  respeta, pero no es lo que recorta un relleno aquí.
- **El modo del lápiz no explica nada aquí.** Implementarlo (§9.7) no movió un píxel: se usa poco y
  casi siempre sobre pantalla en blanco, donde no se distingue de pintar.

#### La trama va del revés, y la línea no era un Bresenham

Con todo lo anterior la diferencia media se quedó en el 0,14 % —1533 píxeles de 1 118 208, sobre 39
capturas—, y ahí se atascó. Parecía el suelo: tramas corridas un píxel de las que no se puede decir
si son nuestras o del recorte de la captura. No lo era. Quedaban dos fallos, y ninguno estaba donde
se buscaba.

**La trama se guarda del revés.** El motor monta las 8 filas del patrón en la pila:

```asm
7E90: dd 21 f8 ff  ld ix,0xfff8
7E94: dd 39        add ix,sp        ; ix = sp - 8, antes de escribir nada
7E96: 06 08        ld b,0x08
7E98: 1a           ld a,(de)        ; fila del UDG
7E99: b6           or (hl)
7E9A: a9           xor c            ; c = 0xFF si la orden pedía invertirla
7E9D: f5           push af
7E9E: 33           inc sp           ; un byte por vuelta, pila abajo
7E9F: 10 f7        djnz 0x7e98
```

`push af / inc sp` escribe un byte por vuelta hacia direcciones descendentes, e `ix` se fijó al
fondo de los ocho **antes** de empezar. Así que `(ix+0)` es la última fila del gráfico e `(ix+7)` la
primera. Y el desplazamiento que se le aplica es `y and 7` (0x7EE8, escrito sobre su propio
`ld d,(ix+NN)`), con la `y` de la ROM contando desde abajo. Resultado: la fila del patrón es
`7 - (y and 7)`, que con la `y` al revés sale **derecha en pantalla**. Indexar el patrón con
`y and 7` a secas lo deja volteado en vertical.

Volteado no se ve como un desplazamiento —es otra cosa— y bajó el total de 1533 a 744 píxeles. Los
dibujos 46 y 47 de la parte 1 pasaron de 57 píxeles cada uno a **cero exacto**, que es la
confirmación que hacía falta.

**Y la línea no es un Bresenham.** Aquí había una afirmación equivocada, escrita al ver que el
contorno coincidía con la captura: lo hacía, pero la medida tolera desplazamientos de un píxel y no
distingue una escalera de una diagonal. El `DRAW` llama a `DRAW-LINE` de la ROM (`0x24BA`, desde
`0x7C9B`), que es un DDA sobre el eje largo: **un punto por cada paso de ese eje**, con un paso del
corto metido en el mismo movimiento cuando el error se pasa, y el error abierto a la mitad del eje
largo (`ld a,b / rra`). Una diagonal sale diagonal, un píxel por fila.

Un Bresenham que mueva un eje por paso saca una escalera con un píxel de más en cada esquina. Y la
diferencia no es cosmética: **una escalera es estanca y una diagonal no**, así que los rellenos —de
los que están hechos estos dibujos— se paran en sitios distintos. Eso era exactamente lo que se veía
como «el relleno se queda corto» en unas pantallas y «el relleno se desborda» en otras, y por eso
fallaba en los dos sentidos: el fallo no estaba en el relleno.

| | Píxeles distintos, 39 capturas |
|---|---|
| Trama volteada + línea de 4 conexiones | 1533 |
| Trama corregida, línea de 4 conexiones | 744 |
| Trama corregida, línea de la ROM | **33** |

La diferencia media queda en **0,003 %**. Las 18 capturas de la parte 2 salen **exactas**, píxel a
píxel, y de las 21 de la parte 1 hay 14 exactas. Lo que queda son 20 píxeles en la fila 0 de cuatro
pantallas —la fila de arriba del todo, donde algo se recorta— y 13 sueltos más.

#### Y sirve para otros dos juegos

Las dos correcciones se dedujeron del desensamblado y se midieron contra La Aventura Original, que
es una sola aventura y un solo dialecto. Cozumel es la prueba de que no estaban ajustadas a ella:
otro dialecto —con la forma corta de `DRAW`—, otra distribución de pantalla y otro dibujante.

Cinco capturas, cinco escenas —Náufrago, Plaza, Playa oeste, Taberna y Despacho, que son los dibujos
4, 5, 7, 11 y 21 de la parte 1—, y **las cinco salen exactas**: cero píxeles distintos con la medida
de vecindad, y entre 2 y 19 sueltos con la comparación cruda, todos en bordes de trama. Ninguna de
ellas es un dibujo simple: la Taberna encadena 491 órdenes y llama a otro de 279.

Y luego **El Jabato, nueve capturas de la primera parte**, emparejadas a ciegas contra los 55
dibujos —cada captura contra todos, y se queda el que menos difiere—. Las nueve aciertan, y por
márgenes que no dejan lugar a duda: la celda da 0 píxeles contra el dibujo 3 y 17 063 contra el
siguiente candidato.

| Captura | Dibujo | Vecindad | Cruda |
|---|---|---|---|
| Vestíbulo | 4 | 0 | **0** |
| Corredor | 6 | 0 | **0** |
| Celda vacía | 15 | 0 | **0** |
| Comienzo (la cara) | 56 | 0 | **0** |
| Logotipo AD | 32 | 0 | 1 |
| Celda del fiambre | 3 | 0 | 2 |
| Puerta de la cárcel | 9 | 0 | 2 |
| Otra celda | 14 | 0 | 2 |
| Sala de guardia | 25 | 0 | 6 |

Cuatro son idénticas **píxel a píxel sin tolerancia ninguna**, y las otras cinco suman 13 píxeles
sueltos entre todas, de 229 376.

Tres de ellas dicen algo más que «coincide»:

- **El comienzo confirma el asunto de la ventana** (§9.6). El dibujo 56 es solo la cara, y sale
  exacta hasta la fila 55; los 624 píxeles de debajo son los créditos, que los escribe el juego. Es
  el mismo caso que la pantalla 44 de La Aventura Original, y ahora sale bien por la razón correcta.
- **El corredor distingue entre tres dibujos casi iguales**: 0 píxeles crudos contra el 6, 3 contra
  el 7 y 6 contra el 8. El 6 es la sala base y los otros dos la llaman y le añaden un detalle, así
  que lo que se está validando ahí es la herencia entre dibujos.
- **El logotipo cae en el 32, al que llaman el 39, el 42, el 46 y el 58**: el mismo patrón de alias
  —entradas que solo contienen `CALL 32 / RET`— que ya se vio en la segunda parte de La Aventura
  Original.

Con esto van tres aventuras contrastadas contra el juego, 53 capturas en total, y las tres eligen
el mismo dialecto de longitudes. Lo que confirma, de paso, cosas que estaban deducidas y sin
contrastar: la elección automática de dialecto, el orden de los operandos de `ATTR` en Cozumel, y
que los dibujos llamados se estampan donde deben.

#### Tres trampas al medir esto

Las tres me costaron tiempo, y ninguna es un fallo del trazado:

1. **La paleta es del emulador.** El que hizo estas capturas pinta el amarillo normal como
   `(182,182,0)`; nosotros usábamos `(205,205,0)`. Cualquier comparación de RGB directo da entonces
   casi todo distinto. Hay que normalizar a los ocho colores y el brillo antes de comparar nada.
2. **Las tramas arruinan el píxel a píxel.** Un damero corrido un píxel no coincide en ninguna
   parte aunque las dos imágenes se vean iguales: una pantalla que marcaba un 19 % era idéntica. La
   medida que sirve es contar los píxeles que no coinciden con **ninguno de los nueve vecinos** del
   otro; así una trama desplazada deja de contar y lo que salta es lo que de verdad está mal.
3. **El texto del juego está en la captura.** Los dibujos que ocupan las 22 filas aparecen con el
   texto de la aventura encima, así que por debajo de la fila 112 no hay nada que comparar.

#### Qué escribe el descompilador

En el `gfx/` de cada parte, junto al resto de recursos y repartido en un subdirectorio por tipo de
fichero:

| Fichero | Qué es |
|---|---|
| `plot/NNN.plot` | el listado de órdenes, numerado como lo numera la tabla |
| `png/NNN.png` | el mismo dibujo trazado, en el lienzo entero de 256×176 |
| `png/NNNxE.png` | ampliado E veces, con `--scale E` |
| `chr/udg.bin` | la tabla de `(0xFFF7)` tal cual |
| `chr/udg.png` | esa tabla como atlas |

Las entradas que solo contienen un `RET` no dan fichero: son entradas reales y la tabla las cuenta,
pero un listado de una orden no sirve de nada y el hueco en la numeración ya lo dice.

Dos líneas del listado no son órdenes del motor. `IMAGE` nombra el dibujo, que es lo que referencia
un `CALL`. Y `INHERIT` recoge el estado con el que se entra —padre, tinta, papel, brillo y
posición— para los dibujos que alguien invoca, porque una pieza pensada para estamparse en una
escena no significa nada leída suelta: sus coordenadas y sus colores son los que le dejó quien la
llamó. Se usa al dibujar el fichero por su cuenta y se ignora al dibujarlo dentro de su escena,
donde el padre fija ese estado al correr.

#### Lo que queda

- Quedan **33 píxeles** en total. Veinte de ellos son la misma cosa: la fila 0 —la de más arriba del
  lienzo— de los dibujos 3, 20, 23 y 27 de la parte 1, donde a una cuña le falta la fila superior
  que la captura sí tiene. Huele a cómo se ajusta la `y` al pasarse por arriba: para `MOVE` y `FILL`
  el motor la envuelve en `0x7E12`, restando 176 al pasar y 80 al bajar de cero, mientras que un
  `DRAW` no pasa por ahí y camina con los incrementos crudos desde `COORDS`. Los otros 13 son
  sueltos, uno de ellos el píxel que deja un `LINE 0,0` en el origen.
- Contrastadas con capturas están **La Aventura Original** —39 pantallas de las dos partes—,
  **Cozumel** —5, todas exactas— y **El Jabato** —9, todas exactas—. De Los Templos Sagrados y
  Chichén Itzá solo se sabe que descodifican y que se ven bien, que no es lo mismo; pero después de
  Cozumel, que era el candidato a dar sorpresas por usar otro dialecto, ya no hay motivo concreto
  para esperarlas.

Y una lección de método, que costó dos días: **la medida que tolera el desplazamiento de un píxel
tolera también que una línea sea una escalera en vez de una diagonal**. Es lo que hacía falta para
no contar las tramas corridas, y es lo que dejó pasar un fallo de trazado durante todo ese tiempo,
disfrazado de fallo de relleno. Cuando una medida se estanca en un suelo que se explica solo como
«ruido de la captura», conviene desconfiar del suelo.

### 9.9. El Amstrad CPC: localizado, sin descodificar

Todo lo anterior es del Spectrum. Aquí se decía que en la carga útil del Amstrad la firma del
despachador no aparecía, y de ahí se concluía que su motor estaría ensamblado de otra forma o
llegaría comprimido. Lo primero es cierto y lo segundo no hacía falta: **el motor está ahí, sin
comprimir, y es el mismo diseño**. Lo que hacía falta eran dos correcciones.

#### Por qué la firma no aparecía

El despachador del Spectrum aísla el opcode con `and 7` y lo duplica con `sla a` para indexar la
tabla de saltos. El del CPC se ahorra el desplazamiento:

```asm
1EBE: dd 7e 00     ld a,(ix+0)
1EC1: e6 0e        and 0x0e         ; el opcode va en los bits 1-3, ya duplicado
1EC3: fe 08        cp 0x08
1EC5: ca ca 1f     jp z,0x1fca      ; un opcode con manejador aparte
1EC8: 21 de 1e     ld hl,0x1ede     ; tabla de 8 manejadores
1ECB: 85 6f        add a,l / ld l,a
1ECD: 5e 23 66 6b  ld e,(hl) / inc hl / ld h,(hl) / ld l,e
1ED1: e9           jp (hl)
```

Mismo mecanismo, otra colocación de bits, así que **el opcode de una orden del CPC no está en los
bits 0-2 sino en los 1-3**, y con él se corren los flags de escala, espejo y color. Los ocho
manejadores de El Jabato caen en `0x1EEE`, `0x1F24`, `0x1F4E`, `0x1FA5`, `0x1FCA`, `0x1FE1`,
`0x1FFF` y `0x2061`, contiguos y en orden.

Y en vez de llamar a la ROM del Spectrum llama al firmware: `GRA PLOT ABSOLUTE` (`0xBBEA`),
`GRA MOVE ABSOLUTE` (`0xBBC0`), `GRA LINE RELATIVE` (`0xBBF9`), `GRA MOVE RELATIVE` (`0xBBC3`),
`GRA SET PEN` (`0xBBDE`) y `SCR SET INK` (`0xBC32`). Buscar esas llamadas es lo que localiza el
motor en las cinco aventuras sin saber nada más.

#### El bloque se ancla en 0x9E00, no en el techo de la memoria

Esta es la corrección que abre la puerta. El manejador de `CALL` lee la tabla de dibujos así:

```asm
1F8B: ed 5b f1 9d  ld de,(0x9df1)
```

Y nadie escribe nunca esa dirección, exactamente igual que en el Spectrum: el puntero viaja con el
bloque. Ahora bien, `0x9E00 − 0x9DF1 = 15`, **el mismo desplazamiento** que `0x10000 − 0xFFF1`. O
sea que la cola es idéntica —tabla de dibujos, fin de tabla, fin de propiedades y UDG en −15, −13,
−11 y −9— y lo único que cambia es dónde acaba el bloque. En el CPC no puede acabar en el techo
porque de `0xC000` para arriba está la pantalla.

Cambiando solo el ancla, la búsqueda de §9.5 encuentra los bloques:

| Edición | Bloques | Dibujos |
|---|---|---|
| La Aventura Original | 2 | **49 y 108** |
| Los Templos Sagrados | 2 | 60 y 57 |
| Chichén Itzá | 2, por duplicado en el disco | 21 y 38 |
| Cozumel | 2 | 31 y 34 |
| El Jabato | ninguno | — |

Los 49 y 108 de La Aventura Original son **exactamente los del Spectrum**, y las longitudes de los
dibujos coinciden en 18 de los 49 de la primera parte y en 62 de los 108 de la segunda. Es el mismo
arte recodificado, no los mismos bytes.

#### Lo que falta para descompilarlos

Nada de esto se ha llevado al descompilador, y no es un cambio de una línea. Falta:

1. **La codificación de las órdenes**, que hay que sacar desensamblando los ocho manejadores: con el
   opcode corrido un bit, las longitudes y el significado de los operandos están sin confirmar.
2. **El color.** El CPC no tiene atributos por celda de 8×8: tiene plumas y una paleta. El modelo de
   pantalla de §9.6 no le sirve.
3. **El lienzo.** El firmware trabaja en coordenadas de usuario de 0-639 × 0-399 sea cual sea el
   modo, mientras que el modo 1 son 320×200 píxeles reales. La escala no es la del Spectrum.
4. **El relleno.** El CPC no tiene un `FILL` en la ROM, así que el motor trae el suyo y hay que
   encontrarlo y portarlo. Después de lo que costó el del Spectrum (§9.8), esta es la parte que más
   cuidado pide.
5. **El Jabato no aparece**, y de camino salió otra cosa: la dirección de carga que leemos de la
   cabecera AMSDOS de `JABATO.B01` está 12 bytes corrida. El desensamblado solo cuadra con `0x0811`
   y la cabecera declara `0x0805`. Puede ser cosa nuestra al reconstruir el fichero, y si lo es
   afecta a más que a los gráficos.

En conjunto son prácticamente un segundo renderizador. Queda anotado y de momento nos quedamos con
los vectoriales del Spectrum, que están cerrados.

#### El PCW no lleva vectores: lleva mapas de bits

El otro Amstrad es otra historia, y más corta. Sus discos llevan cinco cosas:

| Fichero | Qué es |
|---|---|
| `PARTE00N.DDB` | la base de datos |
| `PARTE00N.DAT` | 111–140 KB de ilustraciones |
| `PARTE00N.CHR` | el juego de caracteres, 2176 bytes |
| `PARTE000.PIC` | 16 000 bytes, la pantalla de presentación |
| `AD.COM` | el intérprete, 8960 bytes de CP/M |

Un bloque vectorial entero del Spectrum ronda los 30 KB para 49 dibujos. Aquí hay 111 KB para 51
ilustraciones, con cabecera de tamaño en cada una. **Son mapas de bits**, y encima monocromos, que
es lo que cabía esperar de una máquina de 720×256 sin color.

**`PARTE000.PIC` ya se puede leer.** Son 16 000 bytes con la disposición de la máquina: ocho bytes
por celda de carácter, que son sus ocho líneas de barrido, 80 celdas de ancho y 25 de alto. O sea
640×200, y sale la ilustración de portada de Cozumel firmada por C. Marqués, la misma que en el
Spectrum pero tramada en blanco y negro. Leída como un mapa lineal no sale más que ruido.

**`PARTE00N.DAT` es un archivo de ilustraciones de los que ya sabemos leer**, o casi. Lleva la misma
tabla de 256 ranuras indexada por localidad, con un desplazamiento de 4 bytes por ranura, pero el
registro mide **10 bytes** en vez de los 44 o 48 de las otras máquinas: no hay paleta que guardar.
La cabecera son 6 bytes, así que la primera ilustración cae en 6 + 256×10 = **2566**, que es
exactamente donde empieza en los cuatro discos. Salen 51 y 54 ilustraciones en Cozumel, y 69 y 45 en
Los Templos Sagrados.

Y cada ilustración abre con las mismas tres palabras que en las demás plataformas: bandera y ancho,
alto, y una tercera que aquí es **la longitud del cuerpo** —comprobada contra la distancia a la
ilustración siguiente, y cuadra al byte en todas las que se han mirado—. Las de Cozumel miden
512×104; las de Los Templos van de 16×8 a 128 de ancho.

Lo que **no** está resuelto es cómo se empaquetan los píxeles. No es el esquema que ya
implementamos: se probaron 1, 2 y 4 bits por píxel, con y sin la máscara de colores que dispara las
repeticiones, y ninguna combinación consume la longitud declarada. Habrá que sacarlo de `AD.COM`,
que con 8960 bytes es un blanco pequeño.

### 9.10. No hay herramienta que consultar

El editor que creó estos vectores no está en la distribución de 1991, que es ya el sistema
posterior de mapa de bits. `Deprecated/TOOLS` solo tiene compiladores (`DSC`, `DSA`, `DCA`, `DCS`,
`DSM`), conversores de juegos de caracteres (`CST*`) y `DMG.EXE`, que es el gestor de gráficos **de
PC**. El `S-Pic.doc` de `Docs/` es una herramienta de Amiga para IFF ILBM, ajena a esto. El editor
vectorial era interno de Aventuras AD y no se distribuyó: por eso todo lo de esta sección sale del
intérprete.
