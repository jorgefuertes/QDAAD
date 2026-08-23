# 04 — Condactos

Un condacto es la unidad de ejecución de DAAD: un byte de opcode seguido de sus parámetros.
Este documento fija la tabla completa, la codificación binaria y las aridades irregulares.

La tabla canónica del compilador está en `work/DRC/src/UCondacts.pas:25-172`. Las tablas de los
5 intérpretes coinciden con ella; ver [08-interpretes.md](08-interpretes.md).

---

## 1. Codificación binaria

```text
+--------+-----------+-----------+-----------+
| opcode |  param 1  |  param 2  |  param 3  |
+--------+-----------+-----------+-----------+
   1 B     0 o 1 B     0 o 1 B     0 o 1 B
```

- El opcode ocupa los **bits 0 a 6**; el rango útil es 0–127.
- **El bit 7 del opcode indica indirección del primer parámetro** (`drb.php:1131`):
  ```php
  if (($condact->NumParams>0) && ($condact->Indirection1)) $opcode = $opcode | 0x80;
  ```
  `AT 5` se codifica `00 05`; `AT @5` se codifica `80 05`, y el intérprete toma el contenido del
  flag 5 como número de localidad.
- **El número de parámetros no aparece en el binario**: se deduce del opcode. Un DDB es por tanto
  imposible de recorrer sin la tabla de aridades. Cualquier desincronización desplaza la lectura
  del resto del bloque.
- Fin de bloque: `0xFF`, o bien un condacto terminal (§4).

El bit 7 es la razón de que el formato admita solo 128 condactos, y de que DAAD v3 no pudiera
añadir opcodes nuevos: tuvo que reutilizar huecos ya existentes.

---

## 2. Tabla completa (opcodes 0–127)

Columna **P** = número de parámetros. Columna **T**: **C** = condición (`CanBeJump: true`),
A = acción.

Una *condición* devuelve cierto o falso; si falla, la ejecución abandona la entrada actual y
salta a la siguiente. Una *acción* continúa siempre, salvo las terminales.

| # | Condacto | P | T | Tipos de parámetro |
|---|---|---|---|---|
| 0 | `AT` | 1 | **C** | locno |
| 1 | `NOTAT` | 1 | **C** | locno |
| 2 | `ATGT` | 1 | **C** | locno |
| 3 | `ATLT` | 1 | **C** | locno |
| 4 | `PRESENT` | 1 | **C** | objno |
| 5 | `ABSENT` | 1 | **C** | objno |
| 6 | `WORN` | 1 | **C** | objno |
| 7 | `NOTWORN` | 1 | **C** | objno |
| 8 | `CARRIED` | 1 | **C** | objno |
| 9 | `NOTCARR` | 1 | **C** | objno |
| 10 | `CHANCE` | 1 | **C** | percent |
| 11 | `ZERO` | 1 | **C** | flagno |
| 12 | `NOTZERO` | 1 | **C** | flagno |
| 13 | `EQ` | 2 | **C** | flagno, value |
| 14 | `GT` | 2 | **C** | flagno, value |
| 15 | `LT` | 2 | **C** | flagno, value |
| 16 | `ADJECT1` | 1 | **C** | vocabularyAdjective |
| 17 | `ADVERB` | 1 | **C** | vocabularyAdverb |
| 18 | `SFX` | 2 | A | value, value |
| 19 | `DESC` | 1 | A | locno |
| 20 | `QUIT` | 0 | A | — |
| 21 | `END` | 0 | A | — |
| 22 | `DONE` | 0 | A | — |
| 23 | `OK` | 0 | A | — |
| 24 | `ANYKEY` | 0 | A | — |
| 25 | `SAVE` | 1 | A | value |
| 26 | `LOAD` | 1 | A | value |
| 27 | `DPRINT` | 1 | A | flagno |
| 28 | `DISPLAY` | 1 | A | value |
| 29 | `CLS` | 0 | A | — |
| 30 | `DROPALL` | 0 | A | — |
| 31 | `AUTOG` | 0 | A | — |
| 32 | `AUTOD` | 0 | A | — |
| 33 | `AUTOW` | 0 | A | — |
| 34 | `AUTOR` | 0 | A | — |
| 35 | `PAUSE` | 1 | A | — |
| 36 | `SYNONYM` | 2 | A | vocabularyVerb, vocabularyNoun |
| 37 | `GOTO` | 1 | A | locno |
| 38 | `MESSAGE` | 1 | A | mesno |
| 39 | `REMOVE` | 1 | A | objno |
| 40 | `GET` | 1 | A | objno |
| 41 | `DROP` | 1 | A | objno |
| 42 | `WEAR` | 1 | A | objno |
| 43 | `DESTROY` | 1 | A | objno |
| 44 | `CREATE` | 1 | A | objno |
| 45 | `SWAP` | 2 | A | objno, objno |
| 46 | `PLACE` | 2 | A | objno, locno |
| 47 | `SET` | 1 | A | flagno |
| 48 | `CLEAR` | 1 | A | flagno |
| 49 | `PLUS` | 2 | A | flagno, value |
| 50 | `MINUS` | 2 | A | flagno, value |
| 51 | `LET` | 2 | A | flagno, value |
| 52 | `NEWLINE` | 0 | A | — |
| 53 | `PRINT` | 1 | A | flagno |
| 54 | `SYSMESS` | 1 | A | sysno |
| 55 | `ISAT` | 2 | **C** | objno, locno |
| 56 | `SETCO` | 1 | A | objno |
| 57 | `SPACE` | 0 | A | — |
| 58 | `HASAT` | 1 | **C** | value |
| 59 | `HASNAT` | 1 | **C** | value |
| 60 | `LISTOBJ` | 0 | A | — |
| 61 | `EXTERN` | 2 | A | value, value |
| 62 | `RAMSAVE` | 0 | A | — |
| 63 | `RAMLOAD` | 1 | A | flagno |
| 64 | `BEEP` | 2 | A | value, value |
| 65 | `PAPER` | 1 | A | value |
| 66 | `INK` | 1 | A | value |
| 67 | `BORDER` | 1 | A | value |
| 68 | `PREP` | 1 | **C** | vocabularyPrep |
| 69 | `NOUN2` | 1 | **C** | vocabularyNoun |
| 70 | `ADJECT2` | 1 | **C** | vocabularyAdjective |
| 71 | `ADD` | 2 | A | flagno, flagno |
| 72 | `SUB` | 2 | A | flagno, flagno |
| 73 | `PARSE` | 1 | A | value |
| 74 | `LISTAT` | 1 | A | locno |
| 75 | `PROCESS` | 1 | A | procno |
| 76 | `SAME` | 2 | **C** | flagno, flagno |
| 77 | `MES` | 1 | A | mesno |
| 78 | `WINDOW` | 1 | A | window |
| 79 | `NOTEQ` | 2 | **C** | flagno, value |
| 80 | `NOTSAME` | 2 | **C** | flagno, flagno |
| 81 | `MODE` | 1 | A | value |
| 82 | `WINAT` | 2 | A | value, value |
| 83 | `TIME` | 2 | A | value, value |
| 84 | `PICTURE` | 1 | A | value |
| 85 | `DOALL` | 1 | A | locno |
| 86 | `MOUSE` | 2 | A | value, value |
| 87 | `GFX` | 2 | A | value, value |
| 88 | `ISNOTAT` | 2 | **C** | objno, locno |
| 89 | `WEIGH` | 2 | A | objno, flagno |
| 90 | `PUTIN` | 2 | A | objno, locno |
| 91 | `TAKEOUT` | 2 | A | objno, locno |
| 92 | `NEWTEXT` | 0 | A | — |
| 93 | `ABILITY` | 2 | A | value, value |
| 94 | `WEIGHT` | 1 | A | flagno |
| 95 | `RANDOM` | 1 | A | flagno |
| 96 | `INPUT` | 2 | A | value, value |
| 97 | `SAVEAT` | 0 | A | — |
| 98 | `BACKAT` | 0 | A | — |
| 99 | `PRINTAT` | 2 | A | value, value |
| 100 | `WHATO` | 0 | A | — |
| 101 | `CALL` | 2 | A | value, value |
| 102 | `PUTO` | 1 | A | locno |
| 103 | `NOTDONE` | 0 | A | — |
| 104 | `AUTOP` | 1 | A | locno |
| 105 | `AUTOT` | 1 | A | locno |
| 106 | `MOVE` | 1 | A | flagno |
| 107 | `WINSIZE` | 2 | A | value, value |
| 108 | `REDO` | 0 | A | — |
| 109 | `CENTRE` | 0 | A | — |
| 110 | `EXIT` | 1 | A | value |
| 111 | `INKEY` | 0 | A | — |
| 112 | `BIGGER` | 2 | **C** | flagno, flagno |
| 113 | `SMALLER` | 2 | **C** | flagno, flagno |
| 114 | `ISDONE` | 0 | **C** | — |
| 115 | `ISNDONE` | 0 | **C** | — |
| 116 | `SKIP` | 1 | A | skip |
| 117 | `RESTART` | 0 | A | — |
| 118 | `TAB` | 1 | A | value |
| 119 | `COPYOF` | 2 | A | objno, flagno |
| 120 | `dumb` | 0 | A | — |
| 121 | `COPYOO` | 2 | A | objno, objno |
| 122 | `dumb` | 0 | A | — |
| 123 | `COPYFO` | 2 | A | flagno, objno |
| 124 | `dumb` | 0 | A | — |
| 125 | `COPYFF` | 2 | A | flagno, flagno |
| 126 | `COPYBF` | 2 | A | flagno, flagno |
| 127 | `RESET` | 0 | A | — |

Tipos de parámetro (`UCondacts.pas:8-13`), usados por el análisis semántico del frontend:
`locno` localidad, `objno` objeto, `flagno` flag, `sysno` mensaje de sistema, `mesno` mensaje de
usuario, `procno` proceso, `value` valor libre 0–255, `percent` porcentaje, `vocabulary*` palabra
del vocabulario del tipo indicado, `skip` desplazamiento de salto, `string` cadena literal,
`window` ventana 0–7, `bitno` bit 0–15.

**Conjunto completo de condiciones**: 0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 55, 58, 59, 68, 69, 70, 76, 79, 80, 88, 112, 113, 114, 115.

Las 3 entradas `dumb` — **120, 122 y 124** — son huecos reservados. En DAAD v3 2 pasan a ser
condactos reales y la tercera se usa desde el backend; ver §5 y [07-daad-v3.md](07-daad-v3.md).

---

## 3. Aridades irregulares

Hay 4 casos en los que el número de bytes consumidos no coincide con la aridad de la tabla.
Un intérprete que los ignore se desincroniza y ejecuta basura.

| Caso | Comportamiento |
|---|---|
| `INPUT` (95) | PCDAAD y jDAAD declaran `numParams: 21` en su tabla (`PCDAAD/condacts.pas:260`, `jdaad.js:339`). No son 21 parámetros: es un valor centinela que dispara un tratamiento especial. |
| `EXTERN` (61) con segundo parámetro `3` | Es la codificación de XMESSAGE en DAAD v2 vía Maluva y consume **3 bytes de parámetro**: `offsetLo, 3, offsetHi` (`drb.php:854-863`). NextDAAD lo trata en el recorredor de condactos, no en el manejador (`engine.asm:450-464`), y advierte de que no se escriba `EXTERN n 3` a mano. |
| `SFX` (18) con segundo parámetro 3 o 4 | En PCDAAD consume **un byte extra** del flujo con la frecuencia de muestreo (`PCDAAD/condacts.pas:1590-1600`). En el fuente se escribe con un `#defb` a continuación. |
| Condacto de depuración `0xDC` | Lo inserta DRC en las builds con `-d`. Solo NextDAAD lo salta (`engine.asm:381-397`). En cualquier otro intérprete `0xDC & 0x7F = 92 = NEWTEXT`, lo que destruiría la orden compuesta pendiente. |

---

## 4. Condactos terminales

Estos condactos terminan el bloque por sí mismos. El compilador, fuera de modo clásico, se ahorra
el `0xFF` final cuando el bloque acaba en uno de ellos (`drb.php:1050`):

`DONE` (22), `OK` (23), `NOTDONE` (103), `REDO` (108), `SKIP` (116), `RESTART` (117)

Un intérprete debe detectar el fin de bloque por ambos caminos.

---

## 5. Pseudo-condactos (128–143)

No existen en el binario. Son nombres que acepta el fuente `.DSF` y que el compilador traduce a
otra cosa. `NUM_FAKE_CONDACTS = 16`.

| # | Nombre | P | Traducción |
|---|---|---|---|
| 128 | `XMES` | 1 | **v3**: opcode 120 con 2 parámetros (offset bajo, offset alto). **v2**: `EXTERN offsetLo, 3, offsetHi` vía Maluva (`drb.php:838-866`) |
| 129 | `XMESSAGE` | 1 | El frontend lo convierte en `XMES` añadiendo `#n` a la cadena (`USintactic.pas:633`) |
| 130 | `XPICTURE` | 1 | **Obsoleto**: error de compilación (`drb.php:1015`) |
| 131 | `XSAVE` | 1 | **Obsoleto** (`drb.php:1031`) |
| 132 | `XLOAD` | 1 | **Obsoleto** (`drb.php:1035`) |
| 133 | `XPART` | 1 | **Obsoleto** (`drb.php:1039`) |
| 134 | `XPLAY` | 1 | Cadena MML expandida a una secuencia de `BEEP` (`drb.php:912-947`) |
| 135 | `XBEEP` | 2 | **Obsoleto** (`drb.php:1043`) |
| 136 | `XSPLITSCR` | 1 | **v3**: `GFX x 15`. **v2**: `EXTERN x 6`. Solo CPC y C64 (`drb.php:995-1012`) |
| 137 | `XUNDONE` | 0 | **v3**: error, deprecado (`drb.php:878`). **v2**: `EXTERN 0 7` |
| 138 | `XNEXTCLS` | 0 | **Obsoleto** (`drb.php:1019`) |
| 139 | `XNEXTRST` | 0 | **Obsoleto** (`drb.php:1023`) |
| 140 | `XSPEED` | 1 | **Obsoleto** (`drb.php:1027`) |
| 141 | `PENDINGSKIP` | 1 | Interno del compilador: marca un `SKIP` con etiqueta hacia adelante, resuelto en `drf.pas:302` |
| 142 | `XDATA` | 1 | Cadena `"flag,v1,v2,…"` expandida a una cadena de `LET` (`drb.php:957-994`) |
| 143 | `GETKEY` | 0 | **Solo v3**: se emite como `PAUSE 0`. Sin `-v3` es error (`drb.php:955`) |

Fuera de la tabla hay 2 códigos más, en `UConstants.pas`:

- `FAKE_DEBUG_CONDACT_CODE = 220` — información para el depurador de ZEsarUX; se descarta salvo con `-d`.
- `FAKE_USERPTR_CONDACT_CODE = 256` — nunca se emite; solo registra una entrada en `extvec`.

`JUMP` se acepta como sinónimo de `SKIP` (`UCondacts.pas:195`).

---

## 6. Deduplicación de bloques

Fuera de modo clásico, el compilador comparte físicamente las colas idénticas de distintos
bloques de condactos (`drb.php:768-801`, `1052-1104`). 2 entradas pueden apuntar a posiciones
distintas de una misma secuencia de bytes.

Consecuencias prácticas:

- Un desensamblador de DDB no puede asumir que los bloques son disjuntos.
- La aritmética de `SKIP` debe calcularse sobre entradas, no sobre bytes.
- En plataformas con alineación, los puntos de corte compartidos se restringen a posiciones pares
  (`drb.php:1125`).

`-c` en el backend o `#classic` en el fuente desactivan la optimización.

---

## 7. Verificación

**[V]** `LET 100 @200` compilado con `-v3` para ZX/128K produce en el DDB, en el offset 1715:

```text
7A C8   INDIR 200          ; opcode 122, flag 200
33 64 C8   LET 100, 200    ; opcode 51, param1=100, param2 = marcador
4B 03   PROCESS 3
FF      fin de bloque
```

El marcador del segundo parámetro es el propio número de flag; `INDIR` lo sobrescribe en tiempo
de ejecución. Ver [07-daad-v3.md](07-daad-v3.md#3-indirección-del-segundo-parámetro).

**[V]** El mismo fuente sin `-v3` no compila: `Indirection is not allowed in this parameter.`
