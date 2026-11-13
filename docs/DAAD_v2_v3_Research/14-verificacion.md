# 15 — Verificación empírica

Esta documentación no se ha escrito solo leyendo código. La especificación binaria se ha
contrastado contra DDBs reales generados durante la investigación. Este documento explica cómo
reproducirlo y qué encontró.

Todo el material está en `work/_verify/`.

---

## 1. Montar la cadena en macOS o Linux

El compilador es portable pese a estar distribuido como ejecutables de Windows.

```bash
brew install fpc            # 3.2.2, bottle arm64 nativa

cd work/_verify/tmp
cp ../../DRC/src/*.pas ../../DRC/src/*.l .
rm -f lexer.pas_            # variante antigua del lexer, no compila
fpc -Mobjfpc -O2 -FE. -FU. drf.pas
mv drf ../bin/
```

Resultado: **5254 líneas compiladas en 0,8 s, sin errores**. El binario `drf` funciona igual que
el `drf.exe` del kit.

Para el backend basta con el PHP del sistema:

```bash
php ../DRC/src/drb.php zx 128k EN entrada.json salida.ddb -v
```

PHP 8.5.9 lo ejecuta correctamente. La única incidencia es un aviso de obsolescencia por
`utf8_encode()` (`drb.php:1727`), que no impide nada. Conviene tenerlo en cuenta de cara a
PHP 9, donde la función desaparece.

---

## 2. La matriz de pruebas

`work/_verify/build-matrix.sh` compila `TEST.DSF` (TestUnitDAAD, actualizado para v3) para 7
combinaciones, cada una en v2 y v3. Los targets están elegidos **por lo que discriminan**, no por
cubrir el catálogo:

| Target | Qué pone a prueba |
|---|---|
| `zx 128k` (EN) | Dirección base `0x8400`, little-endian, sin alineación |
| `zx 128k` (ES) | Bit de idioma del byte `0x01` |
| `cpc` | Otra dirección base (`0x2880`) |
| `msx2 8_6` | Byte `0x02` codificando modo de vídeo y ancho de charset |
| `pc vga256` | Base 0 **con alineación a word** |
| `amiga` | **Big-endian** |
| `c64` con `-ch` | Cabecera de contenedor con dirección de carga |

Tamaños obtenidos:

```text
zx128k    v2  6641    v3  7030
pcvga     v2  6878    v3  7282
amiga     v2  6878    v3  7282
msx2      v2  6641    v3  7030
c64       v2  6643    v3  7032     (+2 bytes de cabecera de carga)
cpc       v2  6641    v3  7030
zx128kes  v2  6998    v3  7413
```

Los 389 bytes de diferencia entre v2 y v3 **no son del formato**: `TEST.DSF` usa `#ifdef "V3"`
para compilar pruebas adicionales. Ver §4.

---

## 3. El validador

`work/_verify/ddbdump.py` lee un DDB aplicando la especificación de
[02-formato-ddb.md](02-formato-ddb.md) y [03-secciones.md](03-secciones.md), y falla si algo no
cuadra. Comprueba:

- Versión en el rango válido y byte `0x02` correcto.
- Los **12 punteros caen dentro del fichero** tras restar la dirección base.
- `word(0x20) − base` **coincide con el tamaño real del fichero**.
- El vocabulario está formado por entradas de 7 bytes, deofusca a caracteres imprimibles y
  termina en `0x00`.
- Las 4 tablas de mensajes tienen su tabla de índice, y el primer mensaje de cada una deofusca a
  texto y termina en `0xF5`.
- Las conexiones de la localidad 0 terminan en `0xFF`.
- "Objeto inicialmente en" lleva su `0xFF` final; peso y atributos se decodifican coherentemente.
- Las tablas de entradas de **los 8 procesos** terminan en `0x00`.
- La tabla de tokens tiene al menos un token con bit 7 de fin.

`work/_verify/verify-all.sh` lo aplica a toda la matriz:

```text
zx128k-v2       == TODO OK ==      zx128k-v3       == TODO OK ==
zx128kes-v2     == TODO OK ==      zx128kes-v3     == TODO OK ==
cpc-v2          == TODO OK ==      cpc-v3          == TODO OK ==
msx2-v2         == TODO OK ==      msx2-v3         == TODO OK ==
pcvga-v2        == TODO OK ==      pcvga-v3        == TODO OK ==
amiga-v2        == TODO OK ==      amiga-v3        == TODO OK ==
c64-v2          == TODO OK ==      c64-v3          == TODO OK ==
```

**14 de 14 sin un solo fallo.**

---

## 4. Qué encontró la verificación

Cinco cosas que la lectura del código no habría dado por sentadas.

### 4.1 El word `0x20` no es la longitud del fichero

Fue el **único fallo del validador** en la primera pasada, y resultó ser un error de la
documentación tradicional, no del compilador. En `drb.php:2063`:

```php
$fileSize = $currentAddress;// - $baseAddress;
```

La resta está comentada. El campo guarda la **dirección final**, no la longitud. Comprobado en
4 targets:

| Target | `word(0x20)` | Base | Diferencia | Tamaño real |
|---|---|---|---|---|
| ZX 128K | `0x9F76` = 40822 | `0x8400` = 33792 | **7030** | 7030 |
| CPC | `0x43F6` = 17398 | `0x2880` = 10368 | **7030** | 7030 |
| C64 | `0x53F6` = 21494 | `0x3880` = 14464 | **7030** | 7030 (+2 de cabecera) |
| MSX2 | `0x1B76` = 7030 | 0 | 7030 | 7030 |

En los targets de base 0 coinciden, que es exactamente por lo que el error pasa desapercibido.

### 4.2 v3 no cambia el formato

`BLANK_EN.DSF` no usa condicionales de versión. Compilado para ZX/128K en v2 y v3:

```text
tamaños: v2=2038  v3=2038
offsets distintos: [0]
  0x0000: v2=02 v3=03
```

**Un byte.** Es la prueba más contundente de que DAAD v3 es un modo de ejecución, no un formato.

### 4.3 El endianness de Amiga

El DDB de Amiga leído como little-endian produce **13 fallos de validación**; leído como
big-endian, ninguno. Confirma que ST y Amiga invierten el orden de bytes, pese a que la función
que lo decide se llame `isLittleEndianPlatform`.

### 4.4 El desfase de uno en los tokens

Decodificando el mensaje de sistema 0 con el algoritmo del intérprete:

```text
token[0] del compilador = '\x00'   -> ocupa 1 byte
token[1] = ' the '   token[2] = ' you '
getToken(0) del intérprete = ' the '     == token[1] del compilador
STX[0] decodificado: "It's too dark to see anything."
```

Que es literalmente la línea `/0` de `TEST.DSF`. Queda demostrado que el token 0 es un relleno
descartado y que **debe medir exactamente un byte** para no romper PCDAAD. Ver
[03-secciones.md](03-secciones.md#12-referencia-desde-un-mensaje-y-el-desfase-de-uno).

### 4.5 El bug de `-np`

```text
$ php drb.php pc vga256 EN out/pcvga-v3.json out/np-test.ddb -np
Target: PC (VGA256)
$ ls -l out/np-test.ddb
-rw-r--r--  60 bytes
```

**60 bytes: solo la cabecera.** Sin mensaje de error y con código de salida cero. `drb.php:290`
usa `exit` donde debería usar `return`. Cualquier build que use `-np` produce un DDB inservible
que parece haberse construido bien.

---

## 5. Comprobaciones puntuales

### La indirección del segundo parámetro

Se insertó `LET 100 @200` en una copia de `BLANK_EN.DSF`:

```text
$ ./bin/drf zx 128k IND.DSF out/ind-v2.json
295:29:IND.DSF: Indirection is not allowed in this parameter.

$ ./bin/drf zx 128k IND.DSF out/ind-v3.json -v3
out/ind-v3.json generated.
```

Y en el DDB resultante, offset 1715:

```text
7A C8      INDIR 200
33 64 C8   LET 100, 200
4B 03      PROCESS 3
FF         fin de bloque
```

Confirma la mecánica del prefijo `INDIR` descrita en
[07-daad-v3.md](07-daad-v3.md#3-indirección-del-segundo-parámetro), y que el marcador del
segundo parámetro es el propio número de flag.

### La alineación

Los 12 punteros de un DDB de PC/VGA256 son **todos pares**. En el DDB de ZX equivalente hay 3
impares: tokens (34003), conexiones (38707) y vocabulario (33939). La alineación es
efectivamente específica del target.

### El machine ID de PC/VGA256

Un DDB compilado con `pc vga256` sale con machine ID `0x00`, no `0x0D`. Confirma que la
comprobación de `drb.php:1266` cortocircuita antes de llegar a la de `1276`.

### La cabecera de contenedor de C64

Los 2 primeros bytes de un DDB C64 con `-ch` son `80 38`, es decir `0x3880` en little-endian:
exactamente la dirección base de C64.

---

## 6. Reproducir

```bash
cd work/_verify
bash build-matrix.sh     # genera out/*.json y out/*.ddb
bash verify-all.sh       # valida los 14 DDB y compara v2 con v3
python3 ddbdump.py out/zx128k-v3.ddb            # volcado detallado de uno
python3 ddbdump.py out/amiga-v3.ddb --big-endian
python3 ddbdump.py out/c64-v3.ddb --skip 2
```

Ficheros:

| Fichero | Función |
|---|---|
| `bin/drf` | Frontend construido desde el fuente |
| `build-matrix.sh` | Genera la matriz de DDBs |
| `ddbdump.py` | Vuelca y valida un DDB contra la especificación |
| `verify-all.sh` | Valida toda la matriz y compara v2 con v3 |
| `verify-all.log` | Última salida registrada |
| `out/` | JSON, DDB y registros de compilación |
| `tmp/` | Fuentes y objetos del frontend |

---

## 7. Bases de datos ejecutables generadas

`work/_verify/build-games.sh` compila `TEST.DSF` como base de datos v3 lista para cargar en tres
intérpretes distintos, y la deja empaquetada donde cada uno la espera.

| Destino | Artefacto | Tamaño |
|---|---|---|
| MSX2 SCREEN 8 | `msx/DAAD-EN.DDB`, `msx/DAAD-ES.DDB` | 7030 / 7413 B |
| MSX1 | `msx/DAAD-MSX1.DDB` | 7030 B |
| MSX2, disco arrancable | `msx/TESTV3-EN.DSK`, `msx/TESTV3-ES.DSK` | 720 KB |
| PC/DOS VGA 256 | `pc/GAME-EN/`, `pc/GAME-ES/` (`DAAD.DDB` + `PCDAAD.EXE` + `DAAD.FNT`) | 7282 / 7684 B |
| ZX Spectrum Next | `next/GAME-EN.DDB`, `next/GAME-ES.DDB` | 7030 / 7413 B |

Las 7 bases de datos pasan el validador sin fallos.

### Validación cruzada con la herramienta del propio intérprete

`msx2daad/bin/precomp.php` analiza un DDB para decidir qué condactos incluir al compilar el
intérprete. Sobre la base MSX2 generada aquí:

```text
Loaded 'DAAD-EN.DDB' (7030 bytes)
Condacts in use: 91 (not used: 37)
```

Ni `XMES`, ni `INDIR`, ni `SETAT` aparecen entre los deshabilitados: **el propio código de
msx2daad reconoce los 3 condactos v3 en la base generada**.

### El target NEXTDAAD sí se puede construir

Contra lo que sugiere el estado de `master`, el target existe y funciona: basta extraer
`drf.pas` y `drb.php` de la rama `origin/nextdaad` del fork —sin tocar el clon— y reconstruir el
frontend.

```bash
git -C work/DRC-Next show origin/nextdaad:src/drb.php > bin/drb-next.php
# (ídem para los .pas, luego fpc -Mobjfpc -O2 drf.pas)
./bin/drf-next NEXTDAAD TEST.DSF next/test-EN.json -v3
php bin/drb-next.php NEXTDAAD EN next/test-EN.json next/GAME-EN.DDB
```

**[V]** El DDB resultante pasa las 4 comprobaciones que hace `ddb_load` de NextDAAD:
versión 3 (rango aceptado 2–3), `byte1 & 0xF0 = 0xC0` (es decir `DDB_MACHINE_NXD = 0x0C`),
byte `0x02` igual a 95, y tamaño por debajo de 128 KB.

### Sobre el identificador de máquina en PC

**[V]** La base para `pc vga256` sale con machine ID `0x00`, no `0x0D`, por el cortocircuito
descrito en [02-formato-ddb.md](02-formato-ddb.md#22-byte-0x01--máquina-e-idioma). No impide
nada: PCDAAD no valida el identificador de máquina.

---

## 8. Qué NO se ha verificado

Por honestidad sobre el alcance:

- **No se ha ejecutado ningún intérprete.** Las afirmaciones sobre comportamiento en tiempo de
  ejecución provienen de la lectura del código, no de pruebas sobre emulador. La suite de 323
  pruebas de msx2daad es la mejor fuente empírica disponible para eso.
- **No se ha generado ningún medio final** (TAP, DSK, D64, ADF). La cadena de empaquetado exige
  herramientas de Windows. Lo descrito en
  [11-build-plataformas.md](11-build-plataformas.md) procede de la lectura de los scripts.
- **No se ha ejecutado ningún DDB en un intérprete real**, ni siquiera los generados en §7.
- **No se han verificado los formatos de imagen y sonido** contra ficheros reales; proceden del
  manual y del código de los conversores.
- Los bugs de los intérpretes están **confirmados por lectura de código**, con la línea exacta
  citada, pero no reproducidos en ejecución.
