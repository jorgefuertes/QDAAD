# 01 — Panorama: la cadena completa

Qué componentes existen, qué produce cada uno y dónde vive el conocimiento.

---

## 1. La cadena

```text
   .DSF                .json                .DDB              medio final
  fuente   ──drf──▶  intermedio  ──drb──▶  binario  ──▶  TAP / DSK / D64 / ADF / EXE / web
             (1)                   (2)                (3)

  IMAGES/  ──────── SCRMAKER, sc2daad, img2daad, imgwizard ────────▶
  SOUNDS/  ──────── wav2sfx, img2daad, DMG ───────────────────────▶
```

1. **`drf`** — frontend en **Free Pascal**. Analiza el `.DSF`, resuelve símbolos, etiquetas y
   semántica, y emite **JSON**. No escribe ni un byte del DDB.
2. **`drb.php`** — backend en **PHP**. Consume el JSON y **enlaza el DDB binario** para un target
   concreto. Aquí vive todo el conocimiento del layout.
3. Un conjunto de herramientas por plataforma monta el medio final.

Las 2 piezas juntas se llaman **DRC**, DAAD Reborn Compiler, de Uto, con contribuciones de
Jose Manuel Ferrer Ortiz y Natalia Pujol.

**La consecuencia práctica más importante de esta arquitectura**: si buscas dónde se decide un
byte del DDB, la respuesta es siempre `work/DRC/src/drb.php`. Y como el JSON es una frontera
limpia, cualquiera de las 2 mitades se puede sustituir por separado.

---

## 2. Dónde está cada cosa

| Pregunta | Fichero |
|---|---|
| ¿Cómo se disponen los bytes del DDB? | `work/DRC/src/drb.php` (2100 líneas) |
| ¿Qué condactos existen y con qué aridad? | `work/DRC/src/UCondacts.pas:25-172` |
| ¿Cuáles son los límites del formato? | `work/DRC/src/UConstants.pas` |
| ¿Cuál es la gramática del `.DSF`? | `work/DRC/src/USintactic.pas` y `DSF.l` |
| ¿Qué contiene el JSON intermedio? | `work/DRC/src/UJSONExport.pas` |
| ¿Cómo se interpreta un DDB? (referencia legible) | `work/PCDAAD/pcdaad.pas` y `condacts.pas` |
| ¿Cuál es la semántica correcta de v3? | `work/msx2daad/src/daad_condacts.c` |
| ¿Qué diferencias hay entre intérpretes? | `work/NextDAAD/manual/known-differences.md` |
| ¿Cómo se construye el medio final? | los 20 `.BAT` de `work/daad-ready/` |
| ¿Qué cambió en cada versión del kit? | `work/daad-ready/WHATSNEW.TXT` |

---

## 3. Inventario de repositorios

Estado en el momento de la investigación (agosto de 2026):

| Repositorio | Commit | Fecha | Papel |
|---|---|---|---|
| `work/DRC` | `e7bb170` | 2026-08-15 | Compilador de referencia |
| `work/DRC-Next` | `e7bb170` | 2026-08-15 | Fork para ZX Next |
| `work/jDAAD` | `f21ba61` | 2026-07-10 | Intérprete JavaScript |
| `work/msx2daad` | `afdd6d2` | 2026-05-06 | Intérprete MSX2 |
| `work/NextDAAD` | `2a206be` | 2026-08-19 | Intérprete ZX Spectrum Next |
| `work/PCDAAD` | `687ef2b` | 2026-07-03 | Intérprete MS-DOS |
| `work/ZXDAAD128` | `fe714e0` | 2025-04-16 | Intérprete ZX 128K, formato propio |
| `work/TestUnitDAAD` | `33f9535` | 2026-04-03 | Suite de pruebas en `.DSF`, actualizada para v3 |
| `work/daad-ready` | kit versión **B** | 2026-05-15 | Distribución completa: scripts de build, herramientas, manuales |
| `work/incoming` | — | — | `DAAD-READY-B.ZIP`, origen comprimido de `daad-ready` |

### Sobre `DRC-Next`

`master` es **byte a byte idéntico** al upstream. Toda su aportación está en la rama
`origin/nextdaad`: 2 commits, 14 líneas, que añaden el target `NEXTDAAD` con machine ID `0x0C`,
base `0x0000`, 80×32 caracteres, la permutación de `BEEP` al estilo ZX y 64 KB de XMessages.

**Con el `master` de este árbol no se puede generar un DDB que NextDAAD acepte**, porque su
cargador rechaza cualquier nibble de máquina distinto de `0x0C`.

### Sobre `ZXDAAD128`

No es un intérprete de DAAD v3. Escribe versión 3 en la cabecera para marcar **su propio formato
bancarizado**, con cabecera de 58 bytes y salida en varios ficheros `.ADn`. Su backend
`drb128.php` es un fork divergente de `drb.php`. Ver
[02-formato-ddb.md](02-formato-ddb.md#9-una-variante-que-no-es-daad-v3-zxdaad128).

Fue retirado de DAAD Ready en la versión B (`WHATSNEW.TXT:77-79`).

---

## 4. Cronología de DAAD v3

De `work/daad-ready/WHATSNEW.TXT`:

| Versión | Fecha | Hitos |
|---|---|---|
| 0.6 | — | Se añaden Amiga y Atari ST; aparece ZXDAAD128 |
| 0.8 | — | `CH82CHR`, hooks `CUSTOM*.BAT`, WinAPE → CPCE, `checkver` |
| 0.9 | — | SVGA en PCDAAD, teclado virtual en jDAAD, targets Windows y macOS con ADP |
| 0.9.2 | — | Primeros `.ST` y `.ADF` reales |
| 0.9.3 | 2024 | Nuevo intérprete de Spectrum Next |
| A | 2025-10-20 | Sonido WAV en Amiga, PC y ST; MP3 en HTML; DRO/OPL en PC; `PICTURE`/`DISPLAY` en los targets ZX modernos; **`XMESSAGE`, `XUNDONE` y `XSPLITSCR` internalizados**, con lo que Maluva deja de ser necesario en esos targets (~2 KB libres); se eliminan `XPART` y `XSPEED` |
| A1 | 2025-11-17 | ZX 128K usa el banco 0 para imágenes; imágenes comprimidas en +3 |
| A2 | 2026-02-21 | *Color cycling* en PC; FLI pasa de `SFX` a `GFX`; C64 y Plus/4 pasan a disco completo |
| **B** | **2026-05-15** | **Soporte de DAAD v3**, la primera versión nueva de DAAD desde los noventa. Targets experimentales de Linux, Amiga OCS/HAM/AGA y Atari ST/STE/Falcon. Se elimina el target Windows clásico. `flopgen` sustituye a MSA. Se retira el intérprete basado en ZXDAAD128 |

---

## 5. Plataformas soportadas

Del manual `DOC/doc_en.html`:

**8 bits**: Amstrad CPC, Amstrad PCW 8000/9000, Commodore 64/128, Commodore Plus/4,
MSX1 (≥64 KB), MSX2, ZX Spectrum 48K (cinta y DivMMC), ZX Spectrum 128K, ZX Spectrum +3,
ZX Spectrum Next, ZX-Uno.

**16 y 32 bits**: Atari ST, Commodore Amiga, PC/DOS (VGA y SVGA).

**Modernas**: HTML/JavaScript; y como experimentales Windows, macOS, Linux, Amiga OCS/HAM/AGA y
Atari ST/STE/Falcon030 mediante el intérprete ADP.

Qué intérprete de código abierto cubre cada una y con qué grado de soporte de v3:
[08-interpretes.md](08-interpretes.md#3-matriz-de-soporte-de-daad-v3).

---

## 6. El nombre de los binarios de intérprete

En `daad-ready/ASSETS/` los intérpretes siguen la convención `D<máquina>I<E|S>[F]3`:

```text
DS128KE3.BIN     ZX Spectrum 128K, inglés,  v3
DCPCISF3.BIN     Amstrad CPC,      español, v3
dc64ief3.prg     Commodore 64,     inglés,  v3
DSTEI3.PRG       Atari ST,         inglés,  v3
DAAMIGASI3       Amiga,            español, v3
```

**El `3` final indica DAAD v3.** Es la marca más rápida para saber si un binario del kit soporta
la versión 3.

---

## 7. Estado de la cadena en sistemas Unix

Verificado durante esta investigación sobre macOS arm64:

| Componente | Estado |
|---|---|
| `drf` (frontend) | **[V]** Compila y funciona de forma nativa con `fpc -Mobjfpc -O2 drf.pas` |
| `drb.php` (backend) | **[V]** Funciona con PHP 8.5; solo avisa de que `utf8_encode()` está obsoleta |
| Conversores en PHP | Requieren PHP con GD; deberían funcionar |
| Conversores y empaquetadores en `.exe` | Requieren Windows o emulación |
| Scripts `.BAT` | Requieren `cmd.exe` |

Es decir: **el compilador en sí es completamente portable**; lo que ata el kit a Windows es la
cadena multimedia y de generación de medios.

Detalles en [14-verificacion.md](14-verificacion.md).
