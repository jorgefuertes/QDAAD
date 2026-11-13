# 08 — Los intérpretes

Cinco intérpretes de código abierto, cinco arquitecturas distintas, cinco grados de soporte de
DAAD v3. Este documento describe cómo carga cada uno el DDB, qué implementa y en qué se desvía.

---

## 1. Inventario

| Intérprete | Plataforma | Lenguaje | Núcleo | Commit analizado |
|---|---|---|---|---|
| **PCDAAD** | MS-DOS 386+, VGA/SVGA | Turbo/Borland Pascal 7 | `pcdaad.pas`, `condacts.pas` (2214 l.), `ddb.pas`, `parser.pas` | `687ef2b` (2026-07-03) |
| **jDAAD** | Navegador | JavaScript + canvas | `jdaad.js` (4607 l.) | `f21ba61` (2026-07-10) |
| **msx2daad** | MSX2/2+ con MSX-DOS 1 | C (SDCC 4.5) + Z80 | `src/daad_condacts.c`, `src/daad/*.c` | `afdd6d2` (2026-05-06) |
| **NextDAAD** | ZX Spectrum Next | Z80 puro | `src/engine.asm` + 3 overlays | `2a206be` (2026-08-19) |
| **ZXDAAD128** | ZX Spectrum 128 / +3 | Boriel ZX BASIC + Z80 | `src/ZXDAAD128.bas` (5270 l.) | `fe714e0` (2025-04-16) |

**Linaje.** msx2daad (Natalia Pujol) es el tronco del que derivan ZXDAAD128 (Cronomantic) —
comparte literalmente los nombres `lsBuffer0/1`, `populateLogicalSentence` y los `#define fXxx` —
y NextDAAD, que cita a PCDAAD y msx2daad como árbitros en sus comentarios
(`engine.asm:344-346`, `overlay1.asm:1370-1382`). jDAAD es un port prácticamente 1:1 de PCDAAD;
lo dice el propio código (`jdaad.js:932-944`).

Esto importa al leer discrepancias: **PCDAAD y jDAAD cuentan casi como una sola voz**, y lo
mismo msx2daad y sus dos derivados. Cuando las dos familias coinciden, la afirmación es sólida.

---

## 2. Carga del DDB

| | PCDAAD | jDAAD | msx2daad | NextDAAD | ZXDAAD128 |
|---|---|---|---|---|---|
| Origen | fichero `DAAD.DDB` | array JS `DDBDATA` incrustado | fichero, `loadFilesBin()` | `GAME.DDB` en páginas de 8 KB | ya en RAM: `.AD0` se carga en `$6000` |
| Tamaño máximo | `0xFFFF` | `0xFFFE` | RAM libre | 64 KB (>128 KB = error) | banco 0 |
| Valida versión | **no** | **no** | sí (2 o 3) | sí (2 o 3) | sí, **solo 3** (su propio formato) |
| Valida byte `0x02` = 95 | no | no | no | **sí** | sí |
| Valida máquina | no | no | no | **sí, solo `0x0C`** | no |
| Punteros | offsets absolutos | offsets absolutos | **rebase al cargar** | offsets, ventana paginada | offsets + número de banco |
| Endianness | LE | LE | LE | LE | LE |

Dos detalles que condicionan a un compilador:

**msx2daad reubica los punteros al cargar** (`daad_init.c:68-71`):

```c
for (i=0; i<12; i++) *(p++) += (uint16_t)ddb;
```

Convierte los 12 punteros de la cabecera en punteros reales de RAM. Después, `getPROCess()`
vuelve a sumar la base porque la tabla de procesos sí contiene offsets.

**NextDAAD rechaza deliberadamente los DDB clásicos de ZX** (base `0x8400`). El comentario de
`nextdaad.inc:193-215` explica el porqué: eliminar la aritmética de rebase le permite usar los
64 KB completos en lugar de 31744 bytes.

> **El identificador de máquina de NextDAAD es provisional.** `DDB_MACHINE_NXD equ $0C` está
> marcado como tal en `nextdaad.inc:210-214`. Y el target `NEXTDAAD` **no está en el DRC de este
> árbol**: vive en la rama `origin/nextdaad` del fork. Su `BUILD.BAT:60-66` aborta si no
> encuentra la cadena `NEXTDAAD` en el backend. Con el DRC de `master` no se puede producir un
> DDB que NextDAAD acepte — pero **sí extrayendo `drf.pas` y `drb.php` de la rama del fork**, lo
> que se ha hecho y verificado: ver [14-verificacion.md](14-verificacion.md#el-target-nextdaad-sí-se-puede-construir).

---

## 3. Matriz de soporte de DAAD v3

| Rasgo | PCDAAD | jDAAD | msx2daad | NextDAAD | ZXDAAD128 |
|---|---|---|---|---|---|
| Detección de versión | sí | sí, **pero rota** (§6) | `-DDAADV3` | sí | n/a |
| **120 `XMES`** | ✅ | ✅ | ✅ | ✅ | ❌ |
| **122 `INDIR`** | ✅ parchea el DDB | ✅ parchea el DDB | ✅ parchea | ✅ **sin parchear** (override de un uso) | ❌ |
| **124 `SETAT`** | ✅ | ✅ | ✅ | ✅ | ❌ |
| Banco alt. en `HASAT`/`HASNAT` | ⚠️ **sin comprobar versión** | ❌ no implementado | ✅ | ✅ | ❌ |
| Flag 53 bit 0 (`DOALLNONE`) | ✅ | ✅ | ✅ | ✅ | ❌ |
| Flag 53 bit 4 (`PREPFIRST`) | ✅ | ✅ | ✅ | ✅ | ❌ |
| Flag 53 bit 5 (`UNRECWRD`) | ✅ | ✅ | ✅ | ✅ | ❌ |
| Flag 53 bit 2 (`NOPRONOUN`) | ✅ | ⚠️ **bit equivocado** | ✅ | n/a | ❌ |
| `PAUSE 0` → `GETKEY` | ✅ | ✅ | ✅ | ✅ | ❌ |
| `SYNONYM` sin `done` en v3 | ⚠️ nunca marca `done` | ⚠️ nunca marca `done` | ✅ | ✅ | ⚠️ |
| Maluva desactivado en v3 | ✅ | ⚠️ roto | ✅ | mantiene vectores | ❌ |
| Opcode v3 sobre base v2 | se ejecuta y no hace nada | ídem | consume args y retorna | **error 5** | `NOT_USED` |

**Ranking de completitud v3:** msx2daad ≈ NextDAAD > PCDAAD > jDAAD ≫ ZXDAAD128 (ninguno).

Traducido a decisiones de compilador: si el objetivo es v3 con semántica correcta, los targets
de referencia son **MSX2 y ZX Spectrum Next**. PC/DOS y HTML sirven pero arrastran divergencias
conocidas. ZX 128K vía ZXDAAD128 no es una opción para v3.

---

## 4. Qué NO implementa cada uno

| Intérprete | Sin implementar |
|---|---|
| **PCDAAD** | `CALL` ("PENDING", `condacts.pas:1875-1879`) |
| **jDAAD** | `CALL` ("not supported by jDAAD", `jdaad.js:3952`) |
| **msx2daad** | `CALL`, `MOUSE`; `GFX` parcial; `INPUT` parcial; `SAVE`/`LOAD` sin pedir nombre (lista de TODOs en `daad_condacts.c:240-249`) |
| **NextDAAD** | Enclíticos españoles; artículos españoles; `GFX 9/10/15` son no-op |
| **ZXDAAD128** | `GFX`, `MOUSE`, y todos los condactos v3. Maluva `XPART`, `XBEEP`, `XSPLITSCR`, `XNEXTCLS`, `XNEXTRST`, `XSPEED` |

---

## 5. Peculiaridades por intérprete

### PCDAAD

- Memoria: `{$M 49152,0,458752}` — heap de 448 KB, pila de 48 KB, "para permitir que corran
  EXTERN grandes" (`settings.inc:6`).
- Opciones: `-s` SVGA, `-nomaluva`, `-exec`, `-ndoall`, `-i<fichero>` para órdenes desde fichero,
  `-d` diagnósticos, `-b` doble buffer, `-log`/`-vlog`/`-vilog`.
- **Consola de diagnóstico integrada** (`parser.pas:249-325`): `+f`, `+o`, `+f<n>`, `+f<n>=<v>`,
  `+o<n>`, `+o<n>=<l>`. Muy útil para depurar un DDB generado por un compilador nuevo.
- Emula Maluva solo para `XMES` (función 3) y `XPART` (4). El truco de `XMES` es notable: carga
  512 bytes del `.XMB`, **salva el área de mensajes de sistema del DDB, escribe el xmensaje
  encima, imprime el mensaje 0 y restaura** (`maluva.pas:26-55`, con 30 líneas de justificación).
- Contrato de `EXTERN` (ver `EXTERN.ASM`): `AL` = parámetro 1, `BX` = base de flags, `RETF`.

### jDAAD

- Resolución fija de 320×200, fuente de 6×8, **53 columnas × 25 líneas**.
- Paleta CGA/EGA de 16 colores fija.
- La arquitectura re-entrante es su rasgo definitorio; ver
  [05-flujo-ejecucion.md](05-flujo-ejecucion.md#el-caso-especial-de-jdaad).
- `EXTERN` es extensible desde `extern.js`: `externHandlers[n] = fn`.
- Soporta vídeo MP4, detección de móvil y teclado virtual.

### msx2daad

- Código en `0x0180`, datos en 0, **CRT0 propio y sin libc**. El *heap* es un asignador lineal
  sin `free` real ni `realloc`.
- **`precomp.php` analiza el DDB y genera `include/daad_defines.h` con los `DISABLE_*`**, de modo
  que el binario final no contiene los condactos que el juego no usa. Es la respuesta a la falta
  de memoria: el intérprete se adapta a la base de datos.
- 14 variantes de release: EN/ES × SC5/6/7/8/10/12 más transcript.
- `RAM_MAPPER` (experimental) permite 64 KB de textos externos en RAM paginada; sin él,
  `printXMES` lee del `.XDB` con `fseek`/`fread` de 512 bytes.
- Suite de 323 tests sobre openMSX (292 OK, 0 fallos, 31 TODO). Es el intérprete con mejor
  cobertura de pruebas y por eso el mejor árbitro de comportamiento.
- Vectores `EXTERN` implementados: 0 `XPICTURE`, 1 `XSAVE`, 2 `XLOAD`, 3 `XMES`, 7 `XUNDONE`.

### NextDAAD

Es el intérprete mejor documentado del conjunto: manual completo, 5 documentos de referencia
y una lista explícita de diferencias conocidas.

- **Arquitectura de overlays**: 3 ficheros de 140–172 KB paginados. La tabla de despacho
  `cdisp` guarda para cada condacto la pareja `[página][dirección]` (`engine.asm:850-998`).
- `cprops` (`engine.asm:758-848`) codifica en un byte por condacto: **bit 7 = acción** (ignora el
  acarreo y sella `done` **antes** del despacho), **bit 6 = acción que no sella `done`** (solo
  `SKIP` y `REDO`), **bits 0–1 = número de argumentos**. `QUIT`, `MOVE`, `PICTURE`, `PARSE`,
  `SAVE`, `LOAD` y `SYNONYM` están tipados como condición a propósito.
- **Salta el marcador de depuración `0xDC` de DRC** (`engine.asm:381-397`). Es el único
  intérprete donde una build con `-d` se juega igual que una normal; en los demás,
  `0xDC & 0x7F = 92 = NEWTEXT`.
- **XBN**: binario externo de ≤16 KB con cabecera de 10 bytes y una tabla de servicios congelada
  en `$BEC8` con 10 entradas (`SVC_VERSION`, `PUTCHAR`, `PUTS`, `FOPEN`, `FREAD`, `FWRITE`,
  `FSEEK`, `FCLOSE`, `RANDOM`, `GETMSG`). Contrato de `EXTERN`: `A`/`B` = parámetro 1,
  `C` = función, `HL` = `flags+A`, `DE` = `objTable + A*6`, `IX` = `$A200`.
- **La tabla de objetos es de 6 bytes por objeto**, no un array plano de localidades. Rompe los
  externs clásicos portados.
- Ventana de texto en tilemap, superficie distinta de Layer 2: **máximo 128 combinaciones de
  tinta y papel simultáneas**.
- `INK`/`PAPER`/`BORDER` aceptan 0–255 (extensión propia; jDAAD hace módulo 16).
- `RAMLOAD n` restaura los flags **0 a n inclusive**; jDAAD restaura 0 a n−1.
- `END` confirma contra el mensaje de sistema 31 y `QUIT` contra el 30; jDAAD prueba `END` contra
  el 30.
- `AUTOG` busca aquí → llevado → vestido; jDAAD prueba vestido antes que llevado.

### ZXDAAD128

- **8 binarios precompilados**: TAPE/PLUS3 × 32/42 columnas × EN/ES, seleccionados por su propio
  backend `drb128.php`.
- ORG obligatorio en `0x6002`; heap por defecto de 2048 bytes. "Game error 9" significa heap
  insuficiente.
- **Distribución en bancos**: MTX, STX, LTX, OTX, charset, xmensajes e imágenes migran a otros
  bancos si no caben en el banco 0. Políticas *first fit* (por defecto) o *best fit* (`-b`).
- `-x n` reserva un banco entero para código del autor, invocable con el falso condacto
  `EXTERN n 100` seguido de `#defb p`: cambia de banco y hace `CALL $C000`.
- Maluva soportado: `XPICTURE`(0), `XSAVE`(1), `XLOAD`(2), `XMESSAGE`(3), `XUNDONE`(7) más el
  100 propio. Los valores ≥ 11 y distintos de 255 quedan libres para el autor.
- Reporta resultado de Maluva en el **flag 20**: bit 7 error, bit 0 propagar a `done`.
- **Reutiliza la cabecera del DDB como buffer de tokens** una vez leída
  (`ZXDAAD128.bas:3405`).
- El intérprete 128K basado en ZXDAAD128 **fue retirado de DAAD Ready** en la versión B
  (`WHATSNEW.TXT:77-79`).

---

## 6. Divergencias que afectan a un compilador

Estas no son curiosidades: cambian el binario que conviene generar.

### 6.1 jDAAD se comporta siempre como v3

`V3CODE()` es un **método** de `DDBClass` (`jdaad.js:553-556`), pero se invoca como propiedad en
las 10 llamadas: `jdaad.js:1431, 1468, 1523, 2456, 2980, 3319, 3571, 3581, 4269, 4280`. Una
referencia a función es siempre *truthy*, así que jDAAD ejecuta la rama v3 incluso con una base
v2. Consecuencias:

- `jdaad.js:3319` — `if (!DDB.V3CODE)` nunca se cumple, así que **la emulación de `XMES` vía
  Maluva no se ejecuta jamás**.
- `jdaad.js:2980` — `PAUSE 0` siempre espera tecla.
- `jdaad.js:2456` — `XMES` siempre interpreta el parámetro 2 como byte alto sin leer el byte
  extra del flujo, lo que **desincroniza los condactos en una base v2**.

Que `DDB.isSpanish()` sí se llame con paréntesis en los cinco sitios donde aparece confirma que
es un descuido y no un patrón.

**Implicación**: para HTML, generar siempre `-v3`. Una base v2 no es fiable en jDAAD.

### 6.2 Umbral de nombre convertible: 20 frente a 40

msx2daad usa 20; los demás, 40. Ver
[06-parser.md](06-parser.md#41-nombre-convertible-en-verbo). Un vocabulario con nombres
convertibles entre 20 y 39 se comporta distinto en MSX2.

### 6.3 `HASAT` sin comprobar versión en PCDAAD

Ver [07-daad-v3.md](07-daad-v3.md#4-setat-y-el-banco-alternativo-de-atributos).

### 6.4 `SYNONYM` y `done`

PCDAAD y jDAAD nunca marcan `done`; msx2daad y NextDAAD sí en v2 y no en v3. Un DDB que dependa
del `done` tras `SYNONYM` no es portable.

### 6.5 Flags 29 y 62 en ZXDAAD128

Nunca se escriben, luego `HASAT GMODE` siempre falla. Ver
[05-flujo-ejecucion.md](05-flujo-ejecucion.md#5-flags-29-y-62-detección-de-plataforma).

---

## 7. Tablas de condactos: dónde está cada una

| Intérprete | Ubicación | Forma |
|---|---|---|
| PCDAAD | `condacts.pas:158-289` | `array of (nombre, rutina, numParams)` |
| jDAAD | `jdaad.js:218-370` | array de objetos |
| msx2daad | `daad_condacts.c:27-58` | `const CONDACT_LIST condactList[]` |
| NextDAAD (props) | `engine.asm:758-848` | `cprops`, 1 byte por condacto |
| NextDAAD (despacho) | `engine.asm:850-998` | `cdisp`, página + dirección |
| ZXDAAD128 (despacho) | `ZXDAAD128.bas:3486-3512` | `ON x GOTO` con 128 etiquetas |
| ZXDAAD128 (`ISDONE`) | `ZXDAAD128.bas:2460+` | `condactFlagList(0 TO 127)` |
