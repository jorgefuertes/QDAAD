# 03 — Secciones de datos del DDB

Continuación de [02-formato-ddb.md](02-formato-ddb.md). Aquí se describe el contenido de cada
sección apuntada desde la cabecera.

Convenciones: *word* significa 2 bytes con el orden de bytes del target (little-endian salvo
Atari ST y Amiga). Las marcas **[V]** señalan lo verificado empíricamente.

---

## 1. Tabla de tokens (compresión de texto)

Puntero en `0x08`. Vale `0x0000` cuando no hay compresión (`drb.php:146-150`, `1985`).

### 1.1 Formato en el fichero

Una secuencia de tokens concatenados, **sin contador y sin terminador**. Cada token es una
cadena de caracteres en la que **el último byte lleva el bit 7 activo** (`drb.php:230-232`):

```php
$shift = ($i == $tokenLength-1) ? 128 : 0;
writeByte($outputFileHandler, ord($c) + $shift);
```

La longitud total de la tabla es implícita: el intérprete recorre exactamente tantos
terminadores como necesite. En **modo clásico** la tabla se rellena con espacios hasta
exactamente 128 entradas (`drb.php:202-206`); fuera de él tiene solo los tokens que se usan.

### 1.2 Referencia desde un mensaje, y el desfase de uno

Esta es la parte que más se malinterpreta. Hay **2 numeraciones distintas**.

El compilador sustituye el token `j` de su tabla por el byte `127 + j` (`drb.php:218`), *antes*
de ofuscar el mensaje:

```php
$newMessage = implode(chr($j+127), $parts);
```

El intérprete hace lo contrario (`PCDAAD/messages.pas:93-94`): tras deofuscar, un byte de valor
≥ 128 es una referencia a token, y su identificador es

```pascal
TokenID := (AByte XOR OFUSCATE_VALUE) - 128;
```

Y para localizar ese token **arranca un byte después del puntero** (`PCDAAD/tokens.pas:19`):

```pascal
Ptr := DDBHeader.tokenPos + 1; {Apparently, token table starts one byte after the token pointer}
```

La razón es que **el token 0 del compilador nunca se usa**: es un relleno de un solo byte que
existe solo para que la numeración cuadre. Los juegos de tokens integrados de DRC empiezan con
un token nulo. Así que:

```text
token j del compilador   ↔   TokenID (j−1) del intérprete   ↔   byte (127+j) en el texto plano
```

**[V]** En un DDB de ZX/128K: el token 0 del compilador es `'\x00'` y ocupa exactamente 1 byte;
el token 1 es `' the '`; y `getToken(0)` del algoritmo del intérprete devuelve `' the '`.
Decodificando el mensaje de sistema 0 con este algoritmo se obtiene
`"It's too dark to see anything."`, que es literalmente la línea `/0` de `TEST.DSF`.

> **Trampa de portabilidad.** PCDAAD salta el primer token con un `+1` fijo, lo que asume que
> mide un byte. `msx2daad` lo hace bien, saltando hasta el primer terminador
> (`daad_init.c:73-74`):
> ```c
> while ((*(char*)(hdr->tokensPos++) & 0x80)==0);
> ```
> Con los juegos de tokens estándar el resultado es el mismo. Un compilador que emitiese un
> token 0 de más de un byte rompería PCDAAD y no msx2daad. **Emite siempre un token 0 de un
> solo byte.**

Un detalle adicional: al expandir un token, PCDAAD sustituye cada `_` por un espacio
(`tokens.pas:33`). Es un resto de bases de datos antiguas donde los tokens no podían contener
espacios literales.

### 1.3 Selección de tokens

El compilador hace 2 pasadas (`drb.php:163-196`). En la primera calcula cuánto ahorra cada
token: la primera aparición cuesta 1 byte de definición y cada aparición posterior ahorra
`longitud − 1`. **Los tokens que no se amortizan se descartan**, salvo el token 0, que nunca se
elimina (`drb.php:182`).

Hay juegos de tokens integrados para ES, EN, PT, DE y FR (`drb.php:135-139`). Se pueden
sustituir colocando un fichero `.TOK` con el mismo nombre base que el JSON de entrada
(`drb.php:1765-1772`, `1882-1898`).

### 1.4 Qué tablas se comprimen

`drb.php:310-319`:

| Modo | Tablas comprimidas |
|---|---|
| `basic` | LTX |
| `advanced` | LTX, MTX, STX, XTX |

**OTX (textos de objeto) nunca se comprime**, porque se emite antes de que exista la tabla de
tokens (ver el orden de emisión en [02-formato-ddb.md](02-formato-ddb.md#3-orden-de-emisión-de-las-secciones)).

---

## 2. Tablas de mensajes: MTX, STX, LTX, OTX

Las 4 comparten la misma rutina, `generateMessages` (`drb.php:525-594`), y por tanto el
mismo formato.

### 2.1 Los textos

Cada mensaje se escribe con **todos sus bytes XOR `0xFF`** (`OFUSCATE_VALUE`, `drb.php:244`,
aplicado en `drb.php:540`) y termina con el byte `0x0A` también ofuscado, es decir **`0xF5`**
(`drb.php:551`).

Como `0x0A` está reservado para terminar, los saltos de línea dentro del texto se codifican con
`0x0D`: las secuencias `#n` y `#r` del fuente se traducen a `0x0D` (`drb.php:351`).

**[V]** Los 4 primeros mensajes de cada tabla, en todos los DDB de la matriz, deofuscan a
texto legible y terminan en `0xF5`.

### 2.2 La tabla de índice

Después de los textos se escribe una **tabla de índice**: **un word por mensaje** con la
dirección de cada uno (`drb.php:562-566`). En el código de DRC se la llama *lookup*.

**El puntero de la cabecera apunta a la tabla de índice, no a los textos.** Se calcula hacia
atrás desde la posición actual (`drb.php:2003` para MTX):

```php
$currentAddress - 2 * sizeof(...)
```

Para leer el mensaje *n*:

```text
direccion_texto = word( offset(puntero_de_cabecera) + 2*n )
```

### 2.3 Codificación de caracteres

**No es ASCII.** Es un juego de 256 glifos propio de DAAD, y —esto es lo que suele sorprender—
**es el mismo para todas las máquinas**. Del manual oficial, apéndice A:

> Throughout the whole DAAD system the same character set of 256 characters is used.

El fuente `.DSF` entra en **ISO-8859-1** (ver [12-formato-dsf.md](12-formato-dsf.md)) y el frontend
lo convierte a ese juego en `ConvertChars` (`UJSONExport.pas:157-266`).

#### El reparto del espacio de bytes

Ya deshecho el XOR:

| Valor | Significado |
|---|---|
| `0x00`–`0x09` | literales de control, sin uso |
| **`0x0A`** | **terminador de cadena**; no puede aparecer como dato |
| `0x0B`–`0x0F` | escapes: `#b`, `#k`, `#n`/`#r`, `#g`, `#t` |
| **`0x10`–`0x1F`** | los 16 caracteres latinos frecuentes |
| `0x20`–`0x7F` | ASCII imprimible; el `0x7F` es `ß` |
| `0x80`–`0xFF` | **referencia a token**, con `id = valor − 128` |

Es decir: un carácter literal ocupa `0x00`–`0x7F` y, **ya ofuscado, sale como `0x80`–`0xFF`**; un
token es lo contrario. El backend lo verifica: `checkStrings` (`drb.php:391-416`) aborta si
sobrevive cualquier byte mayor de 127 en MTX, STX, LTX u OTX —antes de ofuscar, claro—.

#### Los 16 códigos bajos

`UJSONExport.pas:170-185`, con el código ISO-8859-1 de origen a la izquierda:

| Código | 16 | 17 | 18 | 19 | 20 | 21 | 22 | 23 |
|---|---|---|---|---|---|---|---|---|
| Carácter | `ª` | `¡` | `¿` | `«` | `»` | `á` | `é` | `í` |

| Código | 24 | 25 | 26 | 27 | 28 | 29 | 30 | 31 |
|---|---|---|---|---|---|---|---|---|
| Carácter | `ó` | `ú` | `ñ` | `Ñ` | `ç` | `Ç` | `ü` | `Ü` |

Es un juego pensado para español y portugués. La copia independiente del backend
(`drb.php:248`, `335-337`) coincide con esta tabla.

> **Trampa.** Los comentarios al final de cada línea de `ConvertChars` **están desplazados en uno a
> partir de `ú`** (dicen `//ú - 26`, `//ñ - 27`…). Los valores que de verdad se emiten son los de
> arriba. Al transcribir, fíate del `\u00XX`, no del comentario.

#### El resto de acentos: `#g` no cambia la codificación

Las demás letras acentuadas —`à ä â è ë ê ì ï î ò ö ô ù û`, todas las mayúsculas acentuadas,
`å ø ð þ ý`— **no caben en el rango bajo**. Se emiten envueltas en un cambio de fuente
(`UJSONExport.pas:186-242`):

```text
#g <n> #t   →   0x0E <n> 0x0F   →   muestra el glifo 128 + n
```

Así `Á` es el glifo 251, `É` el 252, `Í` el 253, `Ó` el 254, `Ú` el 255 y el euro el 224.

**El byte sigue estando entre 0 y 127.** `#g` no codifica nada distinto: le dice al intérprete que
sume 128 al código de cada carácter a la hora de buscar el glifo, hasta que llegue un `#t`. Es un
estado del impresor, no del texto. El condacto `MODE 1` hace lo mismo de forma permanente
(`MODE_FORCEGCHAR`).

`ß` es la excepción: va al rango bajo, al **código 127**, sin envoltorio (`UJSONExport.pas:229`).

#### Las secuencias de escape

`UJSONExport.pas:249-263`, con su equivalente en el backend en `drb.php:351-353`:

| Escape | Byte | Efecto |
|---|---|---|
| `#g` | `0x0E` | pasa a los glifos 128–255 |
| `#t` | `0x0F` | vuelve a los 0–127 |
| `#b` | `0x0B` | **borra la ventana** |
| `#s` | `0x20` | espacio |
| `#k` | `0x0C` | espera una pulsación |
| `#n`, `#r`, `\n`, `\r` | `0x0D` | salto de línea |
| `#f` | `0x7F` | el glifo 127, `ß`. No documentado |
| `#e` | `0E 60 0F` | el euro, glifo 224 |
| `#A`…`#P` | `0x10`…`0x1F` | acceso directo a los 16 códigos bajos |
| `_` | `0x5F` | nombre del objeto, sin artículo |
| `@` | `0x40` | nombre del objeto, artículo en mayúscula. **Solo español** |
| `##` | `#` | almohadilla literal; **solo lo entiende `drb`** (`drb.php:373`) |

> **El manual se equivoca con `#b`.** Dice que imprime un espacio; lo que hace es borrar la ventana
> (`PCDAAD/ibmpc.pas:570`, `msx2daad/src/daad/daad_print.c:40`). El espacio es `#s`.

Las formas antiguas con barra invertida —`\g`, `\t`…— se rechazan con un aviso
(`drb.php:359-368`).

#### Mayúsculas y minúsculas

**Los mensajes se guardan tal cual se escriben.** No hay ningún plegado de caja en todo el camino,
ni ningún target que fuerce mayúsculas: al C64 se le inyecta la fuente de DAAD **encima del juego
de la ROM** (`C64.BAT:102-106`), así que tiene caja mixta como los demás y no usa PETSCII.

El vocabulario es otra cosa: se pasa a mayúsculas y se trunca a 5 (ver §3).

#### La opción `-7`

`ConvertAscii7Chars` (`UJSONExport.pas:45-154`) aplana el texto a ASCII de 7 bits para los targets
que no admiten 8: `ñ`→`ny`, `Ñ`→`NY`, `ß`→`ss`, `þ`→`th`, `«`→`<<`, y las vocales acentuadas a su
letra desnuda. Se aplica a las seis listas de cadenas, **pero no al vocabulario**.

No es una garantía de que todo quede por debajo de `0x7E`: **`-7` sigue emitiendo `0x0E`/`0x0F` y
los códigos 16 a 31** si el fuente usa `#g` o `#A`…`#P` explícitamente. Solo quita los caracteres
Latin-1 del fuente.

### 2.4 Mensajes de sistema forzados a RAM

Los mensajes de sistema `0` a `62` (`LAST_DEFAULT_SYSMESS`, `drb.php:36`) se mantienen siempre
dentro del DDB, incluso con el flag `-x` que desplaza el resto de textos a un fichero externo
`.XMB` (`drb.php:531`). Son los que el intérprete necesita para funcionar antes de poder cargar
nada.

---

## 3. Vocabulario

Puntero en `0x16`. **Entradas de 7 bytes, terminadas por un único byte `0x00`**
(`drb.php:698-709`).

| Offset | Tam | Contenido |
|---|---|---|
| 0 | 5 | La palabra, en mayúsculas, rellenada con espacios hasta 5, **con cada byte XOR `0xFF`** |
| 5 | 1 | Valor de la palabra |
| 6 | 1 | Tipo de palabra |

El relleno es el espacio ofuscado, `0x20 ^ 0xFF = 0xDF`.

### 3.1 Acentos y mayúsculas: las tildes y las eñes se conservan

Una palabra de vocabulario pasa por tres pasos, y el resultado no es el que uno supondría:

1. **Se trunca a 5 caracteres** (`USintactic.pas:305`, `VOCABULARY_LENGTH`). Se trunca, no se
   rechaza.
2. **Se pasa a mayúsculas y después se deshace en los acentos** (`UVocabularyTree.pas:76`):

   ```pascal
   AVocabularyWord := FixSpanishChars(AnsiUpperCase(AVocabularyWord));
   ```

   `FixSpanishChars` (`:39-51`) revierte ocho pares, **de mayúscula a minúscula**:
   `Á→á`, `É→é`, `Í→í`, `Ó→ó`, `Ú→ú`, `Ü→ü`, `Ñ→ñ`, `Ç→ç`.
3. **El backend solo pasa a mayúsculas los bytes entre 32 y 127** (`drb.php:702`), así que los
   códigos 16 a 31 llegan intactos al DDB.

Es decir: **los acentos y la eñe no se quitan, se conservan**. Lo que se normaliza es la caja, y
para los acentuados se normaliza *hacia abajo* porque el juego de caracteres de DAAD no tiene
mayúsculas acentuadas en el rango bajo —`Á` es el glifo 251, al que solo se llega con `#g`— y en el
vocabulario no cabe un escape.

Un ejemplo completo: `araña` y `ARAÑA` acaban las dos como `A R A ñ A`, con la eñe en su código
bajo. La comparación sigue siendo insensible a la caja, que es lo que se buscaba.

> **Un compilador nuevo debe decidir esto conscientemente.** Si normaliza quitando los acentos
> —`araña` → `ARANA`— produce un vocabulario que **no coincide con el de DRC** y, sobre todo, que
> no coincide con lo que el intérprete construye a partir de lo que teclea el jugador: PCDAAD
> convierte una `ñ` tecleada al código 26 (`ibmpc.pas:188`) y solo pliega `a`–`z`
> (`utils.pas:66-70`). Quitar acentos es defendible, pero **hay que quitarlos también en la
> entrada del jugador**, y eso es cosa del intérprete, no del compilador.
>
> Y hay un bug que lo enreda más: `drb` guarda la `ñ` del vocabulario en el código **27** y la de
> los mensajes en el **26** ([13-portabilidad.md](13-portabilidad.md#3-bugs-confirmados-en-el-compilador)).

Tipos (`UVocabularyTree.pas:9`):

| 0 | 1 | 2 | 3 | 4 | 5 | 6 |
|---|---|---|---|---|---|---|
| VERB | ADVERB | NOUN | ADJECTIVE | PREPOSITION | CONJUNCTION | PRONOUN |

2 rangos de valores tienen significado especial:

- **Valor < 14** (`MAX_DIRECTION_VOCABULARY = 13`): palabras de movimiento.
- **Valor ≤ 39** (`MAX_CONVERTIBLE_NAME`): nombres que el parser puede convertir en verbo cuando
  la frase no trae ninguno. Ver [06-parser.md](06-parser.md).

**[V]** El vocabulario de `TEST.DSF` compilado para ZX contiene 9 entradas de 7 bytes; todas
deofuscan a caracteres imprimibles y la tabla termina en `0x00`. Ejemplos leídos del binario:
`'BELOW'` id 151 tipo 4 (preposición), `'CAULD'` id 101 tipo 2 (nombre), `'COAT '` id 102 tipo 2.

---

## 4. Los objetos: 4 tablas independientes

DAAD no tiene un registro de objeto. Reparte los datos en 4 arrays paralelos, cada uno con
su propio puntero de cabecera. Un intérprete que quiera el objeto *n* lo indexa 4 veces.

### 4.1 Nombres — puntero `0x1A`

**2 bytes por objeto** (`drb.php:715-723`): nombre, adjetivo. El valor `0xFF` (`NO_WORD`)
significa "sin palabra".

### 4.2 Peso y atributos — puntero `0x1C`

**1 byte por objeto** (`drb.php:736-754`):

| Bits | Significado |
|---|---|
| 0–5 | Peso (`& 0x3F`, máximo 63) |
| 6 | Es contenedor (`\| 0x40`) |
| 7 | Es vestible (`\| 0x80`) |

El compilador avisa si un objeto contenedor no tiene reservada la localidad de número igual al
suyo (`drb.php:744-749`): los objetos dentro de un contenedor se representan como "estar en la
localidad número igual al identificador del contenedor".

**[V]** En `TEST.DSF` para ZX: obj0 peso 1, obj2 peso 1 vestible, obj3 peso 50 contenedor.
Coincide con la sección `/OBJ` del fuente.

### 4.3 Atributos extra — puntero `0x1E`

**Un word por objeto** (`drb.php:756-764`): los 16 indicadores Y/N de la sección `/OBJ`.

El orden de bits importa y no es evidente. El frontend los lee con
`FOR I := 15 DOWNTO 0` acumulando con `Flags := Flags SHL 1` (`USintactic.pas:483-491`), de modo
que **la columna Y/N más a la izquierda del fuente es el bit 15**.

### 4.4 Localización inicial — puntero `0x18`

**1 byte por objeto, más un `0xFF` final** (`drb.php:725-734`). Valores especiales
(`UConstants.pas:9-13`):

| Valor | Significado |
|---|---|
| 252 | `NOT_CREATED` — no existe todavía |
| 253 | `WORN` — puesto por el jugador |
| 254 | `CARRIED` — llevado por el jugador |
| 255 | `HERE` — en la localidad actual |

**[V]** `TEST.DSF` produce `[254, 252, 253, 1]` seguido de `0xFF`.

---

## 5. Conexiones

Puntero `0x14`. Misma estructura de 2 niveles que los mensajes.

Para cada localidad, una lista de **pares `(dirección, localidad destino)` de 1 byte cada uno,
terminada por `0xFF`** (`drb.php:622-627`). Después de todas las listas, una **tabla de índice
de un word por localidad** (`drb.php:634-638`), que es a lo que apunta la cabecera. El
mecanismo es idéntico al de los mensajes (§2.2).

La dirección es un valor de vocabulario menor que 14 (ver §3).

Toda localidad debe aparecer en la sección `/CON` del fuente, aunque no tenga salidas
(`USintactic.pas:425`); en ese caso su lista es un único `0xFF`.

---

## 6. Procesos

Puntero `0x0A`. Es la estructura más profunda del formato: 3 niveles.

```text
tabla de procesos   →   tabla de entradas   →   bloque de condactos
 (1 word/proceso)        (4 bytes/entrada)       (opcode + parámetros)
```

### 6.1 Bloques de condactos

Un condacto es un **byte de opcode seguido de tantos bytes de parámetro como declare su
aridad**. Los opcodes van de 0 a 127; la tabla completa está en
[04-condactos.md](04-condactos.md).

**La indirección del primer parámetro se codifica activando el bit 7 del opcode**
(`drb.php:1131`):

```php
if (($condact->NumParams>0) && ($condact->Indirection1)) $opcode = $opcode | 0x80;
```

De ahí que el espacio de opcodes esté limitado a 128 y no a 256.

El bloque termina con **`0xFF`** (`END_OF_CONDACTS_MARK`, `PCDAAD/global.pas:12`), emitido en
`drb.php:1190-1194` — pero **solo si el último condacto no era ya terminal**. Fuera de modo
clásico, el compilador reconoce como terminales `DONE`(22), `OK`(23), `NOTDONE`(103),
`SKIP`(116), `RESTART`(117) y `REDO`(108), y en ese caso se ahorra el byte (`drb.php:1050`).
Un intérprete debe tratar el fin de bloque tanto por `0xFF` como por la ejecución de un condacto
terminal.

### 6.2 Tablas de entradas

**4 bytes por entrada** (`drb.php:1200-1216`):

| Offset | Tam | Campo |
|---|---|---|
| 0 | 1 | Verbo — `0xFF` es comodín (el `_` del fuente) |
| 1 | 1 | Nombre — `0xFF` es comodín |
| 2 | 2 | Puntero al bloque de condactos |

La tabla termina con **un solo byte `0x00`** en la posición del verbo
(`END_OF_PROCESS_MARK = $00`, `PCDAAD/global.pas:11`).

> El comentario de `drb.php:1213` dice "doble 00". Es incorrecto: `WriteZero` escribe un byte, y
> tanto PCDAAD como el resto de intérpretes comprueban un único byte. **[V]** La tabla de
> entradas de los 8 procesos de `TEST.DSF` termina en un `0x00` en todos los casos.

### 6.3 Tabla de procesos

**Un word por proceso**, apuntando a su tabla de entradas (`drb.php:1218-1224`). Esto es lo que
apunta el puntero `0x0A` de la cabecera.

**[V]** El DDB de `TEST.DSF` para ZX tiene 8 procesos; el proceso 1 contiene 158 entradas de
4 bytes.

### 6.4 Deduplicación de colas de condactos

Fuera de modo clásico, el compilador calcula un hash de las colas de cada bloque
(`getCondactsHash`, `drb.php:768-801`) y **comparte físicamente los sufijos idénticos entre
entradas distintas** (`drb.php:1052-1078`, `1087-1104`, `1124-1129`). Varias entradas pueden
apuntar a posiciones distintas de una misma región de bytes.

En plataformas con alineación, la compartición se restringe a puntos de corte pares
(`drb.php:1125`).

El flag `-c` del backend, o `#classic` en el fuente, desactiva esta optimización. Un
descompilador o un depurador de DDB tiene que contar con que **los bloques se solapan**.

---

## 7. Externs

Se emiten justo detrás de la cabecera (`drb.php:105-129`). Son bloques binarios crudos
concatenados, incorporados desde el fuente. El sufijo tras `|` determina qué vector de la
cabecera recibe su dirección:

| Directiva del fuente | Vector |
|---|---|
| `#extern` | `extvec[0]` |
| `#sfx` | `extvec[1]` |
| `#int` | `extvec[2]` |
| `#userptr N` (N de 0 a 9) | `extvec[N]` |

`#userptr` es distinto: no incorpora un binario, sino que registra en `extvec[N]` una posición
*dentro de un proceso* (`drb.php:1116-1122`), para que código externo pueda saltar ahí.

---

## 8. Resumen de terminadores

Una tabla que conviene tener a mano al escribir un lector:

| Sección | Terminador |
|---|---|
| Token individual | bit 7 del último byte |
| Tabla de tokens | ninguno (longitud implícita) |
| Mensaje | `0xF5` (`0x0A` ofuscado) |
| Vocabulario | un byte `0x00` |
| Lista de conexiones de una localidad | `0xFF` |
| "Objeto inicialmente en" | `0xFF` tras el último objeto |
| Bloque de condactos | `0xFF`, o un condacto terminal |
| Tabla de entradas de un proceso | un byte `0x00` |
