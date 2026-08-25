# 11 — Generación del medio final por plataforma

Cómo se pasa de un `.DSF` más imágenes y sonidos a un `.TAP`, `.DSK`, `.D64`, `.ADF`, `.ST`,
`.EXE` o sitio web.

Fuente: los 20 ficheros `.BAT` de `work/daad-ready/` y las herramientas de `TOOLS/`. La versión
analizada del kit es la **B (15/05/2026)**, la primera con soporte de DAAD v3
(`WHATSNEW.TXT:4-5`).

---

## 1. El pipeline canónico

Los 20 scripts siguen la misma plantilla de 8 fases. Tomando `ZX128K.BAT` como referencia:

| Fase | Qué hace | Ejemplo |
|---|---|---|
| 1. Config | `CALL CONFIG.BAT` y el hook opcional `CUSTOM.BAT` | `ZX128K.BAT:5-6` |
| 2. Idioma | Elección interactiva; fija `LANG` y `BASELANG` y **los anexa a `CONFIG.BAT`** | `:9-35` |
| 3. Plantilla | Si no existe `%GAME%.DSF`, copia `ASSETS\TEMPLATES\BLANK_%LANG%.DSF` | `:39` |
| 4. **Frontend** | `TOOLS\DRC\DRF <target> [subtarget] %GAME%.DSF` → `%GAME%.json` | `:56` |
| 5. **Backend** | `PHP\PHP TOOLS\DRC\DRB.PHP <target> [subtarget] %LANG% %GAME%.json %GAME%.DDB` → `.DDB` y, si procede, `0.XMB` | `:62` |
| 6. Multimedia | Conversión de `IMAGES/` y `SOUNDS/` al formato nativo | `:73-101` |
| 7. Empaquetado | Intérprete + DDB + fuente + imágenes → medio final | `:106-127` |
| 8. Hooks y emulador | `CUSTOM1.BAT`, limpieza, `CUSTOM2.BAT`, emulador, `CUSTOM3.BAT` | `:129-176` |

**Todos los targets de la versión B invocan `DRF` con `-v3`.** DAAD v3 es el modo por defecto del
kit actual.

### `CONFIG.BAT` — variables globales

```text
:3   PHP\PHP.EXE TOOLS\CHECKVER\checkver.php   comprobación de versión en línea
:5   SET FONT6=AD8x6.CHR        ZX, MSX, PCW, Amiga, ST
:6   SET FONT8=AD8x8.CHR        Amstrad CPC
:7   SET FONTB=C64bold.CHR      C64 y Plus/4
:8   SET FONTPCDAAD=PC.FNT      PC/DOS y targets ADP
:10  SET GAME=TEST              nombre base de todos los artefactos
:12  SET SPLITSCR=splitModeOff  split screen en CPC y C64
:14  SET IMGLINES=96            ALTURA de imagen; afecta a TODOS los targets
:16  SET HIRES=0                1 = SVGA / 640x400 en PC y targets ADP
```

`BASELANG` selecciona el binario del intérprete: `EN` cubre EN, DE y FR; `ES` cubre ES y PT.
Es el mismo criterio que el bit de idioma de la cabecera del DDB.

---

## 2. Tabla maestra

| Script | Frontend | Backend | Imagen | Sonido | Empaquetado | Artefacto |
|---|---|---|---|---|---|---|
| `ZX48.BAT` | `zx 48k -force-normal-messages -v3 tape48` | `zx 48k` | `ZXsplitter.php` → `zx0` → `pager48k.php` | — | `daadmaker` | `RELEASE/ZX48K/TEST.TAP` |
| `ZX128K.BAT` | `zx 128k -v3` | `zx 128k` | `scrcrop` → `zx0` → `SCRMAKER spectrum128` | — | `dd` (font) + `pager128k.php -s` + `daadmaker` | `RELEASE/ZX128K/TEST.TAP` |
| `ZXPLUS3.BAT` | `zx plus3 -v3` | `zx plus3 -3h` | igual + `plus3cache.php` → `DAAD.GRA` | — | `dd` + `SPECFORM -a 63447` + `CPCDiskXP` | `RELEASE/ZXPLUS3/TEST.DSK` |
| `ZXNEXT.BAT` | `zx next -v3` | `zx next` | `SCRMAKER specnext … /s` → `.LY2` | `.aks`, `.WAV` | copia + `daadmaker` | `RELEASE/ZXNEXT/*.LY2` + `.TAP` |
| `ZXUNO.BAT` | `zx uno -v3` | `zx uno` | `SCRMAKER zxuno … 16-63 /SCR` → `.ZXU` | — | copia + `daadmaker` | `RELEASE/ZXUNO/*.ZXU` + `.TAP` |
| `ZXESXDOS.BAT` | `zx esxdos -v3` | `zx esxdos` | `SCRMAKER SPECTRUM` → `.ZX` | — | copia + `daadmaker` | `RELEASE/ZXESXDOS/*.ZX` + `.TAP` |
| `MSX1.BAT` | `msx -v3` | `msx` | `SC2DAAD msx` → `.MS2` | — | `dd` (font) + `dsktool a` | `RELEASE/MSX1/TEST.DSK` |
| `MSX2.BAT` | `msx2 8_6 -v3` | `msx2 8_6` | `imgwizard.php cx … RLE` → `.IM8` | — | font vía `chr2png`+`sc8.php`+`imgwizard`; `dsktool a` | `RELEASE/MSX2/TEST.DSK` |
| `CPC.BAT` | `cpc %SPLITSCR% -v3` | `cpc` | `SC2DAAD cpc` → `.CPC` | — | `MCRF` → `DAAD.BIN` + `CPCDiskXP` | `RELEASE/CPC/TEST.DSK` |
| `CP4.BAT` | `cp4 -v3` | `cp4 -ch` | `SC2DAAD cp4` → `nnn4` | — | `dd` (font) + `c1541` | `RELEASE/CPLUS4/TEST.D64` |
| `C64.BAT` | `c64 %SPLITSCR% -v3` | `c64 -ch` | `SC2DAAD c64` → `nnn64` | — | `dd` (font) + `c1541` | `RELEASE/C64/TEST.D64` |
| `PC.BAT` | `pc vga256 -v3` | `pc vga256` | `SCRMAKER pc … /s` → `.MSD` | `wav2sfx` → `.SFX`; copia `.DRO` y `.FLI` | `XCOPY ASSETS\PC` + `dosbox.conf` | carpeta `RELEASE/PC/` + lanzador DOSBox |
| `PCW.BAT` | `pcw -v3` | `pcw` | — (sin gráficos) | — | `SPECFORM` → `PARTE001.CHR` + `CPCDiskXP` | `RELEASE/PCW/TEST.DSK` |
| `AMIGA.BAT` | `amiga -v3` | `amiga` | `img2daad.php IMAGES;SOUNDS PART1.DAT -a -c` | mismo `img2daad` | `SPECFORM -a 16384`; `PNG2DEGAS`+`PI12BIT`; `exe2adf` | `RELEASE/AMIGA/FLOPPY/TEST.ADF` |
| `ATARIST.BAT` | `st -v3` | `st` | `img2daad.php … -c` | ídem | `SPECFORM`; `PNG2DEGAS`; **`flopgen -s 720`** | `RELEASE/ATARIST/FLOPPY/TEST.ST` |
| `HTML.BAT` | `html -v3` | `html` → **`.JDDB`** | `jDAADImager.php` → `images.js` | `.mp3`/`.mp4` + `jDAADMultimedia.php` | `XCOPY ASSETS\HTML` + `jDAADFontMaker.php` | carpeta `RELEASE/HTML/` |
| `WINDOWS_EXPERIMENTAL.BAT` | `pc vga256 -v3` | `pc vga256` | `ADP\DMG create -format=dat5 -mode=planar8` | dentro del DMG | copia `adp-player.exe` | `RELEASE/WINDOWS_EXPERIMENTAL/TEST.EXE` |
| `MACOS_EXPERIMENTAL.BAT` | `pc vga256 -v3` | `pc vga256` | `DMG create` | ídem | `package_macos_app.php` | `TEST.app` + `TEST-macos.tar.gz` |
| `LINUX_EXPERIMENTAL.BAT` | `pc vga256 -v3` | `pc vga256` | `DMG create` | ídem | AppDir + `package_linux_appdir.php` | `TEST-linux.tar.gz` |
| `AMIGA_EXPERIMENTAL.BAT` | `amiga -v3` | `amiga` | `png2amiga.php` (HAM) + `DMG create -mode=planar5/ham6/planar8` | `-audio-8bit` | `ADP\dsk.exe create -b -r` | `FLOPPY/TEST.ADF` |
| `ATARIST_EXPERIMENTAL.BAT` | `st -v3` | `st` | `DMG create -mode=planar4st/planar8st` | ídem | `ADP\dsk.exe create -d` | `FLOPPY/TEST.ST` |

El target **Windows clásico** (DOSBox + Inno Setup) **fue eliminado** en la versión B
(`WHATSNEW.TXT:29-30`); solo queda el experimental.

---

## 3. Fases no obvias

### 3.1 Inyección del charset con `dd`

El tipo de letra de DAAD se inserta **binariamente** dentro de un fichero contenedor, cortando la
cabecera y la cola con `dd`:

| Target | Contenedor | Cabecera | Cola |
|---|---|---|---|
| ZX 128K | `DAAD.SDG` | `count=13` | `skip=2061 count=28` |
| ZX +3 | `DAAD.SDG` | `count=13` | `skip=2061 count=28`, luego `SPECFORM -a 63447` |
| MSX1 | `DAAD.MDG` | `count=218` | `skip=2266 count=34` |
| C64 | `apart1.prg` | `count=16` | `skip=2064 count=36` |
| Plus/4 | `apart1.prg` | `count=37` | `skip=2085 count=36` |

MSX2 no usa `dd`: convierte el `.CHR` a PNG, luego a SC8, luego a IM8, y lo une con un
`GLUE.IM8`. HTML usa `jDAADFontMaker.php` para producir `font.js`.

### 3.2 ZX 48K: paginación propia

No usa `scrcrop` ni `SCRMAKER`. `ZXsplitter.php <SCR> <líneas>` trocea el SCR según el layout no
lineal del Spectrum; cada trozo se comprime con `zx0`; y `pager48k.php <DDB>` calcula la
dirección de arranque (`0x8400 + tamaño del DDB`, con un mínimo de `0xC000`) y genera `INDEX.BIN`
y `PAGE0.TMP`.

**ZX 48K no admite XMESSAGES**: el script aborta explícitamente (`ZX48.BAT:96-99`). Soporta casi
todas las novedades de v3 **salvo `XMES`** (`WHATSNEW.TXT:21-22`).

### 3.3 XMESSAGES: el fichero `0.XMB`

El backend genera `0.XMB` cuando hay xmensajes (`drb.php:1882-1886`). Cada plataforma lo integra
a su manera:

| Plataforma | Integración |
|---|---|
| ZX +3, PCW, CPC | `CPCDiskXP` al `.DSK` |
| C64, Plus/4 | `c1541`, renombrado a `0nn` |
| MSX2 | **renombrado a `TEXTS.XDB`** |
| Amiga, ST | renombrado a `PART1.XMB` |
| ZX 48K | error: no soportado |

Capacidad máxima por target (`drb.php:420-445`): 2 KB en CPC, C64 y Plus/4; 16 KB en ZX +3, ZX
128K y MSX2; 64 KB en el resto. En +3 y Amiga se reservan además 512 bytes de hueco inicial.

#### El formato del fichero

Es de una simplicidad casi provocadora (`generateXMessages`, `drb.php:448-523`):

```text
[hueco de 512 bytes, solo en +3 y Amiga]
mensaje 0 ofuscado   0xF5
mensaje 1 ofuscado   0xF5
mensaje 2 ofuscado   0xF5
...
[relleno de 512 bytes, solo en Amiga]
```

- **Sin cabecera, sin índice y sin prefijo de longitud.** Mensajes concatenados y nada más. Es la
  diferencia estructural con las tablas de texto del DDB, que sí llevan detrás una tabla de
  punteros de un word por mensaje (`drb.php:562-566`). Aquí no hay nada de eso: la posición de cada
  mensaje la resuelve el compilador y la incrusta en el condacto.
- **Cada byte va en XOR con `0xFF`** (`OFUSCATE_VALUE`, `drb.php:244`), igual que los textos dentro
  del DDB.
- **El separador es un solo byte: `0x0A ^ 0xFF = 0xF5`** (`drb.php:508`). El mensaje siguiente
  empieza justo en el byte de después. Funciona porque un salto de línea dentro del texto se guarda
  como `0x0D`, no como `0x0A` (`drb.php:351`), así que el `0x0A` queda libre para marcar el final.
- Los bytes pueden ser **tokens de compresión**: los xmensajes entran en la tokenización avanzada
  igual que los mensajes normales (`drb.php:316`).

> El comentario de `drb.php:501-502` habla de «guardar la longitud truncada para que quepa en un
> byte». **Está obsoleto**: en el bucle no se escribe ninguna longitud. Un compilador nuevo no debe
> implementarlo.

Cómo lo lee un intérprete, y esto explica el diseño: **no sabe cuánto mide el mensaje**. Lee un
bloque de tamaño fijo desde el offset —511 bytes en PCDAAD (`maluva.pas:66`), 512 en msx2daad
(`daad_platform_msx2.c:367`)— y deja que la rutina normal de impresión se detenga al encontrar el
`0xF5` (`PCDAAD/messages.pas:89`, `msx2daad/src/daad/daad_print.c:143`). PCDAAD llega a copiar esos
512 bytes encima del área de mensajes de sistema del DDB en RAM, imprimir desde allí y restaurar
después la copia de seguridad.

#### Por qué el formato es así

Las tres rarezas del `.XMB` —troceado, relleno y hueco inicial— no son caprichos: cada una
responde a una limitación concreta de un sistema operativo de la época
(`DAAD V3 CAMBIOS.txt:41-43`, y la lógica completa en `generateXMessages`, `drb.php:447-500`).

- **Troceado en varios ficheros.** Hay máquinas cuyo OS carece de `fseek`, así que no se puede
  saltar dentro de un fichero grande. Cuando un mensaje no cabe en el fichero actual, el backend
  lo cierra y abre el siguiente: `0.XMB`, `1.XMB`… En los targets de 2 KB los numera a dos
  dígitos, `00.XMB`, `01.XMB` (`drb.php:455-456`, `484-487`).
- **Relleno hasta el tamaño de bloque.** MSX2, ZX +3 y ZX 128K usan **un solo fichero** y, en vez
  de trocearlo, lo rellenan con ceros hasta completar el bloque (`drb.php:488-492`). Así siempre
  hay al menos un bloque entero que leer desde cualquier offset, y las máquinas que se atragantan
  al leer más allá de EOF no lo hacen nunca.
- **Hueco inicial de 512 bytes** en +3 y Amiga (`drb.php:459-467`). El comentario del código da
  las dos razones, que son distintas: el intérprete de +3 carga los primeros 16 KB en la página 1
  y necesita los primeros 512 bytes como *buffer* para los mensajes que vengan de disco; en Amiga
  es que en máquina real el primer xmensaje se corrompía a veces.
- **Regla de los 512 bytes en +3 y 128K**: además, ningún mensaje puede empezar a menos de 512
  bytes del final del bloque (`$shouldfit512`, `drb.php:474-476`), por el intercambio de 512 bytes
  entre páginas que hace ese intérprete.

#### El offset que viaja en `XMES` es global

Este es el punto que un intérprete debe entender bien. El offset de 16 bits que `XMES` lleva como
parámetros **no es un desplazamiento dentro de su fichero**, sino una dirección lineal sobre la
concatenación de todos ellos (`drb.php:497`):

```php
$GLOBALS['xMessageOffsets'][$i] = $currentOffset + $currentFile * $maxFileSize;
```

Es decir: el intérprete obtiene el número de fichero con `offset / maxFileSize` y la posición
dentro de él con `offset % maxFileSize`, y por eso **necesita conocer el tamaño de bloque de su
propio target**. Como ese tamaño es justo lo que fija la tabla de capacidades de arriba, cambiarlo
rompe todos los offsets ya compilados.

El tope duro es el offset de 16 bits: `Error('Size of xMessages exceeds the 64K limit')`
(`drb.php:844`), sea cual sea el número de ficheros. Ese offset es también **el único límite que
tienen los xmensajes**: no hay ningún contador que los cuente, ni en la cabecera del DDB ni en el
compilador. Ver [15-limites.md](15-limites.md#51-los-xmensajes-no-se-cuentan).

#### Con `-x`, las tablas de texto comparten el mismo fichero

La opción `-x` del backend manda las secciones de texto completas al `.XMB`, y lo hace **añadiendo
al `0.XMB` que ya generaron los xmensajes** (`drb.php:1921-1927`): primero los xmensajes, después
las tablas. Además fuerza el tamaño de bloque a 64 KB en cualquier target (`drb.php:422`), con el
razonamiento de que quien use esto tendrá disco duro, así que en la práctica hay un solo fichero.

Dos secciones no salen nunca del DDB, aunque se pida `-x`:

- **OTX**, los nombres de objeto, porque el motor los necesita en RAM (`drb.php:1935`).
- **Los mensajes de sistema 0 a 62** (`LAST_DEFAULT_SYSMESS`, `drb.php:531`).

Y las tablas de punteros de MTX, STX y LTX **se quedan en el DDB** aunque su texto se vaya: pasan a
contener offsets dentro del `.XMB`. Como siguen siendo de 16 bits, se aplica el mismo techo de
64 KB — pero por ese camino **nadie lo comprueba**. Ver
[13-portabilidad.md](13-portabilidad.md#3-bugs-confirmados-en-el-compilador).

Recordatorio de la cabecera: en ZX con subtarget `PLUS3` y xmensajes presentes, el word `0x20`
del DDB contiene el **tamaño del bloque de XMessages** en lugar de la dirección final. Ver
[02-formato-ddb.md](02-formato-ddb.md#24-word-0x20--no-es-la-longitud-del-fichero).

---

## 4. Inventario de herramientas

### Compilador

| Herramienta | Función |
|---|---|
| `TOOLS/DRC/drf.exe` | Frontend: `.DSF` → `.json` |
| `TOOLS/DRC/drb.php` | Backend: `.json` → `.DDB` |
| `daadmaker.exe` | Construye el `.TAP`: `daadmaker <TAP> <INT> <DDB> [SDG] [SCR] [CHR] [loader] [BIN Index] [/48]`. Identifica los parámetros por extensión |
| `mcrf.exe` | Empaquetador CPC: `MCRF <destino> <intérprete> <DDB> <gráficos> [font]` → `DAAD.BIN` |
| `SCRCROP.exe` | Recorta un SCR de 6912 B a las líneas útiles antes de comprimir |
| `ZXsplitter.php`, `pager48k.php`, `pager128k.php` | Paginación de Spectrum |
| `plus3cache.php` | Genera `DAAD.GRA` desde `SCRMAKER.LOG` |

### Conversión de imagen

`SCRMAKER`, `sc2daad`, `img2daad`, `imgwizard`, `msx-screen-converter`, `jDAADImager`,
`ADP/dmg`, `png2amiga`, `png2falcon`, `png2pcx`, `PNG2DEGAS`, `PI12BIT`. Detalle en
[09-graficos.md](09-graficos.md#4-los-cuatro-conversores).

### Sonido

`WAV2SFX`, `img2daad/wav.php`, `jDAADMultimedia.php`. Detalle en
[10-audio.md](10-audio.md#5-formatos-de-fichero-de-entrada).

### Compresión y medios

| Herramienta | Función | Usada por |
|---|---|---|
| `ZX0/zx0.exe` | Compresor ZX0 de Einar Saukas | ZX 48K, 128K, +3 |
| `DD/dd.exe` | `dd` de GNU para Win32 | inyección de charsets |
| `specform/SPECFORM.EXE` | Añade cabecera de Spectrum/+3; `-a <dir>` fija la dirección de carga | ZX +3, PCW, Amiga, ST |
| `CPCDiskXP.exe` | `-File <f> -AddToExistingDsk <dsk>` | CPC, ZX +3, PCW |
| `dsktool.exe` | `dsktool a <dsk> <fichero>` | MSX1, MSX2 |
| `C1541/c1541.exe` | De VICE: `-attach <d64> -write <src> <destino>` | C64, Plus/4 |
| `exe2adf.exe` | `-d <dir> -l <label> -a <salida.ADF>` | Amiga clásico |
| `FLOPGEN/flopgen.exe` | `-s 720 -o <nombre> <patrones>`. **Sustituye a MSA** desde la versión B | Atari ST clásico |
| `ADP/dsk.exe` | `create -b -r <ADF>` / `create -d <ST>` | targets experimentales |
| `CHECKVER/checkver.php` | Compara la versión local con la publicada; avisa como mucho 4 veces | `CONFIG.BAT` |

### Fuentes de caracteres

- `TOOLS/GCS/` — editor de caracteres SINTAC parcheado para leer y escribir fuentes DAAD.
- `TOOLS/CH82CHR/ch82chr.php <fichero> <tipo>` — convierte fuentes `.ch8` de ZX Origins al
  formato DAAD. Tipos: `6`, `8`, `C64`, `DOSF`, `DOSP`. Los `.ch8` solo traen ASCII 32–127; el
  resto se rellena con la fuente base de `ASSETS/CHARSET`.

Recomendaciones del manual: `AD8x6.CHR` para ZX, MSX, PCW, Amiga y ST — **se muestra a 6 px, hay
que dejar libres las 3 columnas derechas**; `AD8x8.CHR` para CPC; `C64bold.CHR` para C64 y
Plus/4; `PC.FNT` para PC/DOS, que debe guardarse en formato SINTAC.

### `PHP/`

Un runtime PHP 7.4.30 embebido para Windows, incluido para que el usuario no tenga que instalar
PHP. Su `php.ini` activa exactamente dos extensiones: **`gd2`**, imprescindible para todos los
conversores de imagen en PHP, y **`openssl`**, que solo usa `checkver.php`.

En macOS y Linux basta con un PHP del sistema que tenga GD. **[V]** PHP 8.5.9 ejecuta `drb.php`
sin problemas; solo emite un aviso de obsolescencia por `utf8_encode()`.

---

## 5. Estructura de `ASSETS/`

```text
ASSETS/CHARSET/    AD8x6.CHR, AD8x8.CHR, C64bold.CHR, PC.FNT
ASSETS/TEMPLATES/  BLANK_EN|ES|DE|FR|PT.DSF
ASSETS/ZX/         DAAD.SDG + subdirectorios por subtarget
ASSETS/MSX/MSX1/   MSX1.DSK, msxdaad.com, DAAD.MDG, DMSXIEF3.BIN, DMSXISF3.BIN
ASSETS/MSX/MSX2/   MSX2.DSK, GLUE.IM8, msx2daad_3.0.1_EN_SC8.com / _ES_SC8.com
ASSETS/CPC/        BLANK.DSK, BLANK.GRA, DCPCIEF3.BIN, DCPCISF3.BIN
ASSETS/C64/        C64_EN.D64, C64_ES.D64, apart1.prg, dc64ief3.prg, dc64isf3.prg
ASSETS/CP4/        CP4_EN.D64, CP4_ES.D64, apart1.prg, dcp4ief3.prg, dcp4isf3.prg
ASSETS/PCW/        PCW.DSK, DPCWIEF3.BIN, DPCWISF3.BIN
ASSETS/ATARIST/    DSTEI3.PRG, DSTSI3.PRG
ASSETS/AMIGA/      DAAMIGAEI3, DAAMIGASI3
ASSETS/PC/         dosbox.exe, dosbox.conf, GAME/PCDAAD.EXE
ASSETS/HTML/       index.html, jdaad.js, jdaad.css, images.js, extern.js
```

**Convención de nombres de intérprete**: `D<máquina>I<E|S>[F]3`. La `3` final indica **DAAD v3**;
`E` es inglés y `S` español, elegidos por `BASELANG`. Por ejemplo `DS128KE3.BIN` es el intérprete
de ZX 128K en inglés para v3.

`RELEASE/` es solo destino: cada subcarpeta contiene únicamente un `readme.txt` centinela.
`RELEASE/CLEANER.BAT` las vacía.

---

## 6. Reproducir la cadena fuera de Windows

Los `.BAT` son de `cmd.exe`, y varias herramientas son ejecutables de Windows. Pero **el núcleo
del compilador es portable**:

- `drf` es Free Pascal: **[V]** compila y funciona de forma nativa en macOS arm64 con
  `fpc -Mobjfpc -O2 drf.pas`.
- `drb.php` es PHP puro: **[V]** funciona con PHP 8.5 del sistema.

Lo que sí requiere Windows o emulación es la cadena multimedia y de empaquetado (`SCRMAKER`,
`daadmaker`, `MCRF`, `CPCDiskXP`, `dsktool`, `c1541`, `exe2adf`, `flopgen`, `dmg`…), con la
excepción de las herramientas escritas en PHP (`img2daad`, `imgwizard`, `jDAADImager`,
`png2amiga`, `plus3cache`, los pagers), que solo necesitan PHP con GD.

Un compilador nuevo que quiera cubrir la cadena completa en Unix tendría que reimplementar o
sustituir el bloque de empaquetado, no el de compilación.

---

## 7. Errores conocidos en los scripts del kit

Detectados leyendo los `.BAT`; ninguno afecta al DDB, pero sí a los artefactos generados:

| Fichero | Problema |
|---|---|
| `MSX1.BAT:103` | `IF LANG == "EN"` sin `%…%`: la comparación **siempre es falsa**, así que MSX1 siempre coge el intérprete español `DMSXISF3.BIN` |
| `MSX2.BAT:147-148` | La ruta `..\TOOLS\dsktool` está partida en 2 líneas y rompe el bucle de imágenes 100–255 |
| `ZXNEXT.BAT:118` | Comilla desbalanceada en la línea de ZEsarUX |
| `HTML.BAT:98` | Ruta mal formada `IMAGES\\HTML0%%i.mp4` en el `IF exist` |
| `PC.BAT:110-111` | Usa `DELETE`, que no existe en `cmd.exe`; el error se silencia |
| `PC.BAT:204,210` | El mensaje dice `SET SVGA=1` pero la variable real es `HIRES` |
| `C64.BAT:67-68` | `MOVE %GAME%.DDB DAAD.DDB` duplicado |
| `ZX48.BAT:107` | `SET %LOADING=...` con un `%` de más |

Herramientas presentes pero que ningún script invoca: `TOOLS/cut`, `TOOLS/ZIP`,
`TOOLS/PCXFIXER` (comentada en `PC.BAT:129`), `TOOLS/CH82CHR`, `TOOLS/GCS`, `ADP/chr.exe`,
`ADP/ddb.exe`, `specform/MKP3FS.EXE`, `TOOLS/CPC6128H`. Son utilidades manuales o legado.

Además, la tabla de scripts de `DOC/doc_en.html` está **desactualizada**: cita `CPLUS4.BAT`,
`PCDOS.BAT`, `ZX48TAPE.BAT`, `ZX128TAPE.BAT`, `ZX128PLUS3.BAT` y `WINDOWS.BAT`, nombres que ya no
existen.
