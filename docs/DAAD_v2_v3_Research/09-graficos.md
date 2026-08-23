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
