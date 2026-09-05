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

El juego de instrucciones, el modelo de coordenadas y **dónde viven los dibujos en el disco** están
resueltos (§9.1, §9.2 y §9.5). El bloque del Spectrum de La Aventura Original está localizado y sus
157 dibujos —49 en la parte 1 y 108 en la parte 2— descodifican enteros.

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
polilínea, no una lista de segmentos sueltos. `FILL` hace lo mismo — también mueve el cursor
(`0x7ACB`).

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
  atributos** —calcula una dirección en `0x5800` con la fila y la columna de los bytes 3 y 4—.

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

#### Queda pendiente el Amstrad

Todo esto es del Spectrum. En la carga útil del Amstrad la firma del despachador no aparece (§9.4),
así que su motor está ensamblado de otra forma o llega comprimido, y no se le ha aplicado este
método. El mismo orden de trabajo debería servir: mirar primero si hay cintas que declaren las
direcciones de carga, y validar cualquier candidato descodificándolo entero.

#### No hay herramienta que consultar

El editor que creó estos vectores no está en la distribución de 1991, que es ya el sistema
posterior de mapa de bits. `Deprecated/TOOLS` solo tiene compiladores (`DSC`, `DSA`, `DCA`, `DCS`,
`DSM`), conversores de juegos de caracteres (`CST*`) y `DMG.EXE`, que es el gestor de gráficos **de
PC**. El `S-Pic.doc` de `Docs/` es una herramienta de Amiga para IFF ILBM, ajena a esto. El editor
vectorial era interno de Aventuras AD y no se distribuyó: por eso todo lo de esta sección sale del
intérprete.
