# 02 — Formato de la base de datos DDB

Este es el documento central. Describe el fichero `.DDB` byte a byte: cabecera, punteros,
direcciones base, endianness, alineación y límites.

Toda la generación del binario ocurre en un único fichero: **`work/DRC/src/drb.php`**. Ni el
frontend Pascal ni ningún otro componente escriben bytes del DDB. Cuando este documento cita
una línea, cita esa fuente salvo indicación contraria.

Las afirmaciones marcadas con **[V]** están verificadas empíricamente sobre DDBs generados en
esta investigación; ver [14-verificacion.md](14-verificacion.md).

---

## 1. Naturaleza del fichero

Un DDB es una **imagen plana de memoria de como máximo 64 KB, sin reubicación**. No es un
formato con offsets relativos: los punteros internos son **direcciones absolutas del espacio de
direcciones de la máquina destino**. Por eso la dirección base tiene que conocerse en tiempo de
enlazado y por eso un DDB de ZX Spectrum no se puede cargar en otra máquina sin reprocesarlo.

El tope es duro: `drb.php:2074` aborta si la dirección final supera `0xFFFF`.

```text
offset_de_fichero = puntero_del_DDB − direccion_base
```

En los targets con base 0 (PC, ST, Amiga, HTML, MSX2, ZX81/16K) ambas cosas coinciden, lo que
hace muy fácil confundirlas al leer un volcado. No lo son.

---

## 2. Cabecera

La cabecera ocupa **60 bytes (`0x3C`)**: un bloque clásico de 34 bytes (`0x22`) seguido de 26
bytes de vectores extern. Se escribe en `drb.php:1829-1869` con los punteros a cero, y se
reparchea al final en `drb.php:2034-2068`, cuando ya se conocen.

| Offset | Tam | Campo | Referencia |
|---|---|---|---|
| `0x00` | 1 | **Versión DAAD**: `2` o `3` | `drb.php:1833-1835` |
| `0x01` | 1 | **`machineID << 4 \| bit de idioma`** | `drb.php:1838-1842` |
| `0x02` | 1 | Carácter nulo de `/CTL` (**95**) — reutilizado como *submachine ID* en MSX2 | `drb.php:1844-1846` |
| `0x03` | 1 | Número de objetos | `drb.php:1849-1850` |
| `0x04` | 1 | Número de localidades | `drb.php:1852-1853` |
| `0x05` | 1 | Número de mensajes de usuario (MTX) | `drb.php:1855-1856` |
| `0x06` | 1 | Número de mensajes de sistema (STX) | `drb.php:1858-1859` |
| `0x07` | 1 | Número de procesos | `drb.php:1861-1862` |
| `0x08` | 2 | Puntero a la tabla de **tokens** — `0x0000` si no hay compresión | `drb.php:2039` |
| `0x0A` | 2 | Puntero a la **tabla de procesos** | `drb.php:2041` |
| `0x0C` | 2 | Puntero a la **tabla de índice** de textos de objeto (OTX) | `drb.php:2043` |
| `0x0E` | 2 | Puntero a la **tabla de índice** de textos de localidad (LTX) | `drb.php:2045` |
| `0x10` | 2 | Puntero a la **tabla de índice** de mensajes de usuario (MTX) | `drb.php:2047` |
| `0x12` | 2 | Puntero a la **tabla de índice** de mensajes de sistema (STX) | `drb.php:2049` |
| `0x14` | 2 | Puntero a la **tabla de índice** de conexiones | `drb.php:2051` |
| `0x16` | 2 | Puntero al **vocabulario** | `drb.php:2053` |
| `0x18` | 2 | Puntero a **"objeto inicialmente en"** | `drb.php:2055` |
| `0x1A` | 2 | Puntero a **nombres de objeto** (nombre + adjetivo) | `drb.php:2057` |
| `0x1C` | 2 | Puntero a **peso y atributos** de objeto | `drb.php:2059` |
| `0x1E` | 2 | Puntero a **atributos extra** de objeto | `drb.php:2061` |
| `0x20` | 2 | **Dirección final del DDB** (ver §2.4) — o tamaño de XMessages en ZX PLUS3 | `drb.php:2063-2066` |
| `0x22` | 26 | **13 vectores extern** (`extvec[0..12]`) | `drb.php:1866-1869`, `2067-2068` |

> **Qué es una "tabla de índice".** Cinco de estos punteros —`0x0C`, `0x0E`, `0x10`, `0x12` y
> `0x14`— no apuntan a los datos, sino a una **tabla de índice**: un array de words, uno por
> elemento, donde cada word contiene la dirección del elemento correspondiente. Para leer el
> elemento *n*:
>
> ```text
> direccion_del_elemento_n = word( offset(puntero_de_cabecera) + 2*n )
> ```
>
> La indirección existe porque mensajes y listas de conexiones son de **longitud variable**: sin
> ella, llegar al mensaje 174 obligaría a recorrer los 173 anteriores contando terminadores. En
> el código de DRC y en la documentación original a esta tabla se la llama *lookup*.
>
> **[V]** En el DDB de ZX/128K el puntero `0x12` vale `0x882A` (offset 1066). Ahí empieza la
> tabla de índice de los mensajes de sistema: `índice[0] = 0x85A0` → offset 416, donde está
> `"It's too dark to see anything."`; `índice[1] = 0x85B2` → offset 434, `"I can also see: "`.

**No hay checksum en ningún punto del DDB.** El único checksum del ecosistema pertenece a la
envoltura opcional +3DOS (`drb.php:1474-1476`), que es un contenedor, no parte del DDB.

Los 3 campos de conteo de mensajes son bytes: eso fija el techo de 255 en cada tabla.

### 2.1 Byte `0x00` — versión

```php
$b = 2;
if ($v3code) $b = 3;
```

`$v3code` viene del JSON intermedio (`drb.php:1793`), donde lo pone `drf` al recibir `-v3`.
**DRC nunca emite versión 1**; el valor 1 corresponde a bases de datos históricas (Aventura
Original, Jabato) que ningún intérprete moderno genera.

La detección en tiempo de ejecución es una comparación directa. Por ejemplo `PCDAAD/ddb.pas:149`:

```pascal
V3CODE := DDBHeader.version = 3;
```

**[V]** Compilando `BLANK_EN.DSF` para ZX/128K con y sin `-v3`, los 2 DDB resultantes tienen
el mismo tamaño (2038 bytes) y difieren **en un único byte: el offset `0x00`**. Es la prueba
más directa de que v3 no cambia el formato. Ver [07-daad-v3.md](07-daad-v3.md).

### 2.2 Byte `0x01` — máquina e idioma

Nibble alto = identificador de máquina (`getMachineIDByTarget`, `drb.php:1264-1280`):

| ID | Target | | ID | Target |
|---|---|---|---|---|
| `0x00` | PC | | `0x08` | ZX81 |
| `0x01` | ZX Spectrum | | `0x0B` | CP/M |
| `0x02` | C64 | | `0x0C` | **NEXTDAAD** — solo en la rama `nextdaad` del fork |
| `0x03` | Amstrad CPC | | `0x0D` | HTML (jDAAD) |
| `0x04` | MSX | | `0x0E` | Commodore Plus/4 |
| `0x05` | Atari ST | | `0x0F` | MSX2 |
| `0x06` | Amiga | | | |
| `0x07` | Amstrad PCW | | | |

`0x09` y `0x0A` están sin asignar.

Nibble bajo = **bit 0 activo si el idioma es ES o PT** (`drb.php:1840`). DE, FR e EN lo dejan a
cero: el bit no distingue idiomas, distingue **familias de parser** (español vs. inglés).

**[V]** ZX/EN → `0x10`; ZX/ES → `0x11`; MSX2 → `0xF0`; Amiga → `0x60`; C64 → `0x20`;
PC/VGA256 → `0x00`.

> **Trampa.** El código pretende asignar `0x0D` a PC con subtarget VGA256 (`drb.php:1276`),
> pero la comprobación de `PC` en `drb.php:1266` devuelve `0x00` antes de llegar. **[V]** un DDB
> `pc vga256` sale con machine ID `0x00`. En la práctica `0x0D` solo se alcanza vía HTML.

### 2.3 Byte `0x02` — carácter nulo o *submachine ID*

Históricamente contenía el carácter nulo declarado en la sección `/CTL`, que por convención es
el subrayado, ASCII **95**. Por eso todos los DDB clásicos llevan `0x5F` aquí.

`getSubMachineIDByTarget` (`drb.php:1248-1261`) devuelve 95 para todo **excepto MSX2**, donde el
byte se reaprovecha para codificar el modo de vídeo:

```php
$submachineID = $mode - 5;                    // SCREEN 5..12  ->  0..7
if ($charWidth == 8) $submachineID += 128;    // bit 7 = charset de 8 px
```

**[V]** subtarget `8_6` (SCREEN 8, charset de 6 px) produce `0x02 = 3`, exactamente `8 − 5`.

NextDAAD valida este byte y rechaza el DDB si no vale 95 (`NextDAAD/src/nextdaad.inc`, cargador
en `file.asm:222-224`).

### 2.4 Word `0x20` — no es la longitud del fichero

El campo se documenta tradicionalmente como *file length* o *SPARE*. **No contiene la longitud.**
En `drb.php:2063` la resta de la dirección base está comentada:

```php
$fileSize = $currentAddress;// - $baseAddress;
```

Es decir, guarda la **dirección absoluta del final del DDB** en el espacio de la máquina destino.
Para obtener el tamaño real hay que restar la dirección base:

```text
longitud = word(0x20) − direccion_base
```

En targets con base 0 los 2 valores coinciden, y de ahí viene la confusión.

**[V]** ZX/128K: `word(0x20) = 0x9F76 = 40822`, base `0x8400 = 33792`, diferencia **7030**, que
es exactamente el tamaño del fichero. CPC: `0x43F6 − 0x2880 = 7030`. C64: `0x53F6 − 0x3880 =
7030` (más 2 bytes de cabecera de contenedor). MSX2 y PC: coincidencia directa.

**Excepción**: en ZX con subtarget `PLUS3` y con XMessages presentes, este word contiene el
**tamaño del bloque de XMessages**, no la dirección final (`drb.php:2065`). Lo necesita el
soporte de XMessages sobre +3DOS.

### 2.5 Vectores extern (`0x22`–`0x3B`)

13 words. Se rellenan desde `$adventure->extvec[0..12]`, que el frontend alimenta con:

- Bloques `#extern` → `extvec[0]`
- Bloques `#sfx` → `extvec[1]`
- Bloques `#int` → `extvec[2]`
- La directiva `#userptr N` (N de 0 a 9) → `extvec[N]`, apuntando a una posición dentro de un
  proceso (`drb.php:1116-1122`)

Un DDB sin código externo lleva los 26 bytes a cero. **[V]**

---

## 3. Orden de emisión de las secciones

**El orden físico en el fichero no es el orden de los punteros en la cabecera.** El bucle de
emisión (`drb.php:1929-2031`) es:

```text
cabecera (60 B)
  → EXTERNs
  → OTX (textos de objeto)
  → nombres de objeto
  → peso y atributos
  → atributos extra
  → "inicialmente en"
  → VOCABULARIO
  → TOKENS
  → STX (mensajes de sistema)
  → MTX (mensajes de usuario)
  → LTX (textos de localidad)
  → CONEXIONES
  → PROCESOS
```

Los textos de objeto van deliberadamente los primeros para que queden en RAM cuando se usa el
flag `-x`, que desplaza el resto de textos a un fichero externo (`drb.php:1935`).

Un implementador de intérprete **no debe asumir este orden**: debe navegar siempre por los
punteros de la cabecera. Un implementador de compilador sí puede elegir otro orden, siempre que
los punteros sean coherentes.

---

## 4. Direcciones base por target

`getBaseAddressByTarget`, `drb.php:1282-1300`. Redefinible con la opción `-b=` del backend.

| Target | Base | Target | Base |
|---|---|---|---|
| ZX Spectrum | `0x8400` | C64 | `0x3880` |
| Amstrad CPC | `0x2880` | Commodore Plus/4 | `0x7080` |
| MSX | `0x0100` | ZX81 / 16K | `0x0000` |
| Amstrad PCW | `0x0100` | ZX81 / SD81B | `0x8400` |
| CP/M | `0x2000` | NEXTDAAD (fork) | `0x0000` |
| *resto* (PC, ST, Amiga, HTML, MSX2) | `0x0000` | | |

Las bases no nulas corresponden a máquinas donde la zona baja está ocupada por la ROM, el área
de sistema o el propio intérprete.

### C64 y Plus/4: cabecera de dirección de carga

Con la opción `-ch`, el backend antepone al fichero **2 bytes con la dirección de carga en
little-endian** (`prependC64HeaderToDDB`, `drb.php:1493-1509`), tal como espera el cargador de
Commodore. El DDB propiamente dicho empieza en el offset 2.

**[V]** Los primeros bytes de un DDB C64 con `-ch` son `80 38` = `0x3880`, exactamente la
dirección base de C64.

---

## 5. Endianness

Todos los words de un DDB se escriben con la misma rutina, `writeWord` (`drb.php:67-79`):

```php
function writeWord($handle, $word, $littleEndian)
{
    $a = ($word & 0xff00) >> 8;   // alto
    $b = ($word & 0xff);          // bajo
    if ($littleEndian) { $tmp = $b; $b = $a; $a = $tmp; }
    writeByte($handle, $b);
    writeByte($handle, $a);
}
```

Y el selector es:

```php
function isLittleEndianPlatform($target)
{ return (($target=='ST') || ($target=='AMIGA')); }
```

> **Los 2 identificadores están invertidos.** Con `$littleEndian = false` la función escribe
> primero el byte bajo, es decir **little-endian**; con `true` escribe primero el alto, es decir
> **big-endian**. Y `isLittleEndianPlatform` devuelve cierto justamente para las 2 máquinas
> 68000, que son big-endian. Los 2 errores se cancelan y **la salida es correcta**, pero
> cualquiera que porte este código guiándose por los nombres producirá lo contrario de lo que
> pretende. El fork ZXDAAD128 corrigió la nomenclatura: `$isBigEndian = false`
> (`ZXDAAD128/DRC/drb128.php:14`).

La regla real, sin la confusión de nombres:

| Familia | Targets | Orden de bytes |
|---|---|---|
| Z80 / 6502 / 8086 | ZX, CPC, MSX, MSX2, C64, CP4, PCW, CPM, ZX81, PC, HTML | **little-endian** |
| 68000 | Atari ST, Amiga | **big-endian** |

**[V]** El DDB de Amiga leído como little-endian produce 13 fallos de validación; leído como
big-endian, ninguno.

---

## 6. Alineación (*padding*)

`isPaddingPlatform` (`drb.php:1302-1305`) devuelve cierto para **PC, ST, AMIGA y HTML**. En esos
targets se inserta un byte cero cuando hace falta para que cada sección y cada mensaje empiecen
en dirección par (`drb.php:288-296`). Son las plataformas de 16/32 bits, donde el acceso a word
desalineado es caro o ilegal.

**[V]** En un DDB de PC/VGA256 los 12 punteros de la cabecera son pares sin excepción. En el DDB
de ZX equivalente, 3 de ellos son impares (tokens, conexiones, vocabulario).

Las opciones `-p` y `-np` fuerzan y desactivan el relleno.

> **`-np` está roto y falla en silencio.** `drb.php:290` usa `exit` donde debería usar `return`:
>
> ```php
> if ($GLOBALS['adventure']->forcedNoPadding) exit;
> ```
>
> El programa termina en la primera comprobación de alineación. **[V]** `drb.php pc vga256 EN
> … -np` produce un fichero de **60 bytes** — solo la cabecera — sin ningún mensaje de error y
> con código de salida cero. Cualquier build que use `-np` genera un DDB inservible que además
> parece haberse construido bien.

---

## 7. Límites del formato

Todos son incondicionales; **ninguno cambia en v3** (`work/DRC/src/UConstants.pas:18-27`).

| Límite | Valor | Motivo estructural |
|---|---|---|
| Objetos | 256 | contador de 1 byte |
| Localidades | 255 | contador de 1 byte |
| Mensajes por tabla (MTX/STX/LTX) | 255 | contador de 1 byte |
| Procesos | 255 | contador de 1 byte |
| Opcodes de condacto | **128** | el bit 7 del opcode señala indirección |
| Parámetros por condacto | 3 | `MAX_CONDACT_PARAMS` |
| Rango de un parámetro | 255 | 1 byte |
| Longitud de palabra de vocabulario | 5 | `VOCABULARY_LENGTH` |
| Peso de objeto | 63 | 6 bits |
| Tamaño total del DDB | 65535 − base | imagen plana sin reubicación |

El techo de 128 opcodes es la restricción estructural más seria del formato y la razón de que
v3 no pudiera añadir opcodes nuevos: sus 3 condactos nuevos tuvieron que ocupar huecos que ya
existían en la tabla.

La forma real de superar el límite de 255 mensajes **no es v3**, sino el modo no-clásico: el
frontend derrama automáticamente mensajes de MTX a STX y a LTX reescribiendo el opcode
correspondiente (`MES` ↔ `SYSMESS` ↔ `DESC`), lo que da acceso a unas 3 × 255 cadenas
(`UMessageList.pas:76-109`).

---

## 8. Comprobaciones cruzadas independientes

La cabecera de 34 bytes está confirmada por 3 implementaciones que no comparten código con
`drb.php`:

- `work/PCDAAD/ddb.pas:13-34` — el `record TDDBheader` de Turbo Pascal, campo por campo,
  con `sizeof = 34`.
- `work/NextDAAD/src/nextdaad.inc:584-601` — `DDB_HEADER_SIZE equ 34`, y las constantes
  `HDR_NUMOBJ equ 3` … `HDR_OBJEXTR equ 30`.
- `work/msx2daad/include/daad.h:106-134` — el `struct` en C.

---

## 9. Una variante que no es DAAD v3: ZXDAAD128

`work/ZXDAAD128/DRC/drb128.php` es un backend distinto que escribe **versión 3 de forma
incondicional** (`drb128.php:1904`) y máquina `0x01` (ZX), pero **su formato no es DAAD v3**.
Lo que marca es su propio formato bancarizado:

- Cabecera clásica ampliada de `0x22` a **`0x3A` (58 bytes)**, más 16 bytes de paleta y luego
  los 13 vectores extern (`drb128.php:1934`, `2305-2347`).
- Campos nuevos: 4 pares *(offset, número de banco)* para XMessages, offset y banco del
  charset, del índice de imágenes, buffer de objetos, número de imágenes, código de cursor.
- La salida no es un fichero sino **uno por banco**: `.AD0`, `.AD1`, … (`drb128.php:2366`).
- No implementa ningún condacto de v3.

Su validación de carga (`ZXDAAD128/src/ZXDAAD128.bas:3424`) exige versión 3 y byte `0x02` igual
a 95, que es exactamente lo que cumple **cualquier DDB DAAD v3 legítimo**. Un DDB v3 de otro
target pasaría la comprobación y luego se leería con una cabecera de 58 bytes sobre datos de 34,
interpretando texto como números de banco. No hay comprobación del nibble de máquina, pese a que
la constante `MACHINE_SPECTRUM_128 = 16` está definida.

Consecuencia para un compilador universal: **la versión 3 de ZXDAAD128 y la versión 3 de DAAD
son formatos distintos que comparten byte identificador.** Ver [13-portabilidad.md](13-portabilidad.md).

---

## 10. Resumen para implementar un lector

```text
1. Lee 34 bytes. Si version ∉ {2,3}, no es un DDB de DRC.
2. maquina = byte[1] >> 4  →  determina la dirección base y el orden de bytes.
3. Todos los words se leen little-endian, salvo en Atari ST y Amiga.
4. offset_fichero(p) = p − base
5. longitud = word(0x20) − base       (no es la longitud tal cual)
6. Navega siempre por los punteros. Nunca asumas el orden físico de secciones.
7. Los 26 bytes de 0x22 a 0x3B son 13 vectores extern; cero significa "sin vector".
```

Los contenidos de cada sección se describen en [03-secciones.md](03-secciones.md).
