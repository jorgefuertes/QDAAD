# 12 — El fuente `.DSF`

Gramática del fichero que consume el frontend `drf`. Referencias: `work/DRC/src/DSF.l` (el
tokenizador TPLex), `USintactic.pas` (el analizador) y `drf.pas` (línea de órdenes y símbolos).

> Los ficheros `.DSF` están en **ISO-8859-1**, no en UTF-8. `DSF.l` incluye un comentario de
> 5 líneas con caracteres acentuados puesto ahí a propósito para que se note al abrirlo con
> la codificación equivocada.

---

## 1. Estructura general

Las secciones deben aparecer **en este orden exacto** y no es negociable
(`USintactic.pas:934-958`):

```text
/CTL  /VOC  /STX  /MTX  /OTX  /LTX  /CON  /OBJ  /PRO 0 … /PRO n  /END
```

Cada analizador de sección está codificado para terminar cuando encuentra el token de la
sección *siguiente* (por ejemplo `ParseSTX` termina en `T_SECTION_MTX`, `USintactic.pas:382`).
Reordenar las secciones no produce un error claro, sino un fallo de análisis confuso.

El fichero **debe contener `/END`**; `drf` lo comprueba antes de empezar (`drf.pas:417`).

---

## 2. Las secciones

| Sección | Contenido | Sintaxis |
|---|---|---|
| `/CTL` | Carácter nulo | Un único `_`. Es una reliquia: acaba en el byte `0x02` del DDB |
| `/VOC` | Vocabulario | `PALABRA valor tipo`, un tipo de: `verb`, `noun`, `adjective`, `pronoun`, `conjugation`, `preposition`, `adverb` |
| `/STX` | Mensajes de sistema | `/n "texto"` |
| `/MTX` | Mensajes de usuario | `/n "texto"` |
| `/OTX` | Textos de objeto | `/n "texto"` |
| `/LTX` | Textos de localidad | `/n "texto"` |
| `/CON` | Conexiones | `/n` y después pares `DIRECCIÓN destino` |
| `/OBJ` | Objetos | `/n  dónde  peso  c  w  <16 Y/N>  nombre  adjetivo` |
| `/PRO n` | Procesos | Entradas `> verbo nombre` seguidas de condactos |
| `/END` | Fin | — |

Reglas que el compilador impone:

- **La numeración de los mensajes debe ser consecutiva desde 0** en las 4 tablas de texto
  (`USintactic.pas:348`).
- **Toda localidad debe aparecer en `/CON`**, aunque no tenga salidas (`USintactic.pas:425`).
- Varias líneas `>` consecutivas comparten un mismo bloque de condactos.

Ejemplo de `/OBJ`, de `BLANK_EN.DSF:264-270`:

```text
;obj  starts  weight    c w  5 4 3 2 1 0 9 8 7 6 5 4 3 2 1 0    noun   adjective
/0      CARRIED 1       _ _  _ _ _ _ _ _ _ _ _ _ _ _ _ _ _ _    TORCH  _
```

Las 16 columnas de Y/N son los atributos extra. **La columna más a la izquierda es el bit 15**
del word que se emite (`USintactic.pas:483-491`).

---

## 3. Elementos léxicos

De `DSF.l`:

| Elemento | Significado |
|---|---|
| `;` | Comentario hasta el fin de línea |
| `_` y `*` | Comodín o valor nulo |
| `>` | Inicio de entrada de proceso |
| `@` | Indirección |
| `$nombre` | Etiqueta |
| `/nnn` o `/nombre` | Entrada de lista |
| `"…"` | Cadena **o expresión aritmética** |
| `-?[0-9]+` | Número, con signo |

El caso de `"…"` merece una nota: en contextos donde se espera un valor, la cadena se evalúa
como una expresión aritmética mediante `fpexprpars` (`USintactic.pas:108-140`). Por eso los
operandos de `#ifdef` **tienen que ir entrecomillados** (`USintactic.pas:267`).

---

## 4. Directivas del preprocesador

Todas se pueden usar en medio del código; `Scan()` las procesa en línea
(`USintactic.pas:234-281`).

| Directiva | Función |
|---|---|
| `#define nombre [valor]` | Define un símbolo |
| `#ifdef "s"` / `#if "s"` | Compilación condicional |
| `#ifndef "s"` | Condicional negada |
| `#else`, `#endif` | Cierre de condicional |
| `#echo "texto"` | Mensaje en tiempo de compilación |
| `#include "fichero"` | Inclusión. **Solo en el nivel superior y sin anidamiento** (`drf.pas:185`) |
| `#userptr N` | Registra una posición del proceso actual en `extvec[N]`, N de 0 a 9 |
| `#int`, `#sfx`, `#extern` | Incorporan un binario y lo enlazan a `extvec[2]`, `extvec[1]` y `extvec[0]` |
| `#incbin "fichero"` | Incorpora un binario en línea |
| `#hex "…"` | Bytes en hexadecimal |
| `#db` / `#defb` | Byte literal |
| `#dw` / `#defw` | Word literal |
| `#debug` | Activa el modo depuración |
| `#classic` | Desactiva las optimizaciones del backend |

`#defb` es imprescindible para los condactos de aridad irregular: el byte extra de
`SFX n 3` se escribe así. Ver [04-condactos.md](04-condactos.md#3-aridades-irregulares).

---

## 5. Etiquetas y `SKIP`

`$etiqueta` marca un destino para `SKIP` (o su sinónimo `JUMP`).

- Hacia atrás se resuelven en línea (`USintactic.pas:668-679`).
- Hacia adelante se emite el pseudo-condacto `PENDINGSKIP` y se resuelven al terminar el fichero
  con `FixForwardLabels` (`drf.pas:302`, `USintactic.pas:56-93`).

Restricciones: el desplazamiento se mide en **entradas**, con un rango de ±128, y **el salto no
puede salir del proceso** (`USintactic.pas:76`). El máximo de etiquetas es 1024.

---

## 6. Caracteres de escape en los textos

| Escape | Efecto |
|---|---|
| `_` | Nombre del objeto, sin artículo |
| `@` | Nombre del objeto con artículo capitalizado — **solo en bases españolas** |
| `#b` / `#s` | Espacio |
| `#k` | Espera a que se pulse una tecla |
| `#n` / `\n` | Salto de línea (se codifica como `0x0D`) |
| `#r` | Salto de línea |
| `#g` | Cambia al juego de caracteres alternativo |
| `#t` | Vuelve al juego normal |
| `#e` | Símbolo del euro |

---

## 7. Línea de órdenes de `drf`

```text
drf <target> [subtarget] <fichero.DSF> [salida.json] [opciones] [símbolos adicionales]
```

**Targets**: `ZX CPC C64 CP4 CPM MSX MSX2 ZX81 PCW PC AMIGA ST HTML`, más `NEXTDAAD` en la rama
`nextdaad` del fork.

**Subtargets** (obligatorios para ZX, ZX81, MSX2 y PC):

| Target | Subtargets |
|---|---|
| ZX | `48K 128K PLUS3 ESXDOS UNO NEXT` |
| ZX81 | `16K SD81B` |
| MSX2 | `{5,6,7,8,10,12}_{6,8}` — modo de vídeo y ancho del charset |
| PC | `VGA256 VGA EGA CGA TEXT` |

**Opciones** (`drf.pas:355-408`):

| Opción | Efecto |
|---|---|
| `-verbose` | Salida detallada |
| `-no-semantic` | Sin análisis semántico |
| `-semantic-warnings` | Los errores semánticos pasan a avisos |
| `-force-normal-messages` | Todo `XMES`/`XMESSAGE` pasa a mensaje normal. Ver §7.1 |
| `-force-x-messages` | Las cadenas literales de `MES`/`MESSAGE` pasan a xmensajes. Ver §7.1 |
| `-check-maluva-disabled` | No comprueba si Maluva está incluido |
| `-replace-xcondacts` | Sustituye `XSAVE`, `XPICTURE` y `XLOAD` por los originales si el target no los admite |
| **`-v3`** | **Compila para DAAD versión 3** |
| `-7` | Fuerza ASCII de 7 bits en los mensajes |

Las parejas incompatibles se rechazan en `drf.pas:413-414`.

### Línea de órdenes de `drb`

```text
php drb.php <target> [subtarget] <idioma> <entrada.json> [salida.DDB] [opciones]
```

Idiomas: `EN ES DE FR PT`.

| Opción | Efecto |
|---|---|
| `-v` | Detallado, con el desglose de tamaños por bloque |
| `-ch` | Antepone la cabecera de dirección de carga de Commodore (C64 y Plus/4) |
| `-3h` | Antepone la cabecera +3DOS |
| `-c` | Fuerza modo clásico |
| `-d` | Fuerza modo depuración (solo ZX y CPC) |
| `-p` / `-np` | Fuerza o desactiva la alineación. **`-np` está roto**; ver [02-formato-ddb.md](02-formato-ddb.md#6-alineación-padding) |
| `-x` | Manda las **secciones de texto completas** al fichero `.XMB`. Ver §7.1 |
| `-b=` | Redefine la dirección base |

### 7.1 Tres opciones que no hacen lo mismo

Los nombres invitan a confundirlas, y una está en el frontend y la otra en el backend.

En el fuente, un mensaje se puede escribir de dos maneras:

```text
MESSAGE 5                   ; referencia a la entrada 5 de la tabla /MTX
MESSAGE "Coges la llave."   ; cadena literal en línea
```

**`-force-x-messages`** (`drf`) actúa **solo sobre la segunda**. Cuando el analizador encuentra una
cadena literal como parámetro, reescribe el opcode (`USintactic.pas:622-627`):

```pascal
// Implements the ForceXMessages parameter
IF ((Opcode IN [MES_OPCODE, MESSAGE_OPCODE]) OR (Opcode = MES2_OPCODE)) AND (ForceXMessages) THEN
BEGIN
    if Opcode = MES_OPCODE THEN Opcode := XMES_OPCODE
                           ELSE Opcode := XMESSAGE_OPCODE;
END;
```

A partir de ahí sigue el camino normal de los xmensajes: el `XMESSAGE` se convierte en `XMES` con
un `#n` añadido —esa es la única diferencia entre los dos, el salto de línea— y el texto se inserta
en la lista `XTX`, que acaba en el `.XMB`, en lugar de en la tabla MTX que vive dentro del DDB. Un
`MESSAGE 5` no se entera de nada, como avisa la propia ayuda (`drf.pas:35`): *«Does not affect
those written in the MTX table»*.

**Para qué sirve: para recuperar RAM.** Los mensajes de MTX viven dentro de la imagen de 64 KB del
DDB; los xmensajes, en disco y bajo demanda. Y hay una segunda razón, más importante de lo que
parece: **la lista de xmensajes no tiene el límite de 255** que sí tienen las tablas del DDB. Ver
[15-limites.md](15-limites.md#51-los-xmensajes-no-se-cuentan).

El precio es depender de disco, y un tope de 511 caracteres por mensaje (`USintactic.pas:631`).

**`-force-normal-messages`** (`drf`) hace lo contrario: convierte todo `XMES`/`XMESSAGE` en
mensajes normales, para targets sin disco o sin Maluva. Las dos son **mutuamente excluyentes**;
`drf.pas:414` aborta si se pasan juntas.

**`-x`** (`drb`) es otra cosa: se lleva las **secciones de texto enteras** —MTX, STX y LTX— al
`.XMB`, no solo las cadenas escritas en línea. OTX nunca sale del DDB, y los mensajes de sistema 0
a 62 tampoco. Ver
[11-build-plataformas.md](11-build-plataformas.md#33-xmessages-el-fichero-0xmb).

Resumiendo la diferencia: `-force-x-messages` afecta a lo que el autor escribe entre comillas
dentro de un condacto; `-x` afecta a las tablas completas.

---

## 8. Símbolos definidos automáticamente

`drf` define estos símbolos antes de analizar el fuente (`drf.pas:233-285`), lo que permite
escribir código condicional por plataforma:

| Grupo | Símbolos |
|---|---|
| Plataforma | El nombre del target (`zx`, `msx2`…), el subtarget y `MODE_<subtarget>` |
| Tamaño de palabra | **`BIT8`** en ZX, CPC, PCW, MSX, C64, CP4, MSX2, ZX81 y CPM; **`BIT16`** en PC, Amiga y ST. **HTML no recibe ninguno** (`drf.pas:243`) |
| Pantalla | `COLS`, `ROWS` |
| Objetos | `NOT_CREATED=252`, `WORN=253`, `CARRIED=254`, `HERE=255`, `LAST_OBJECT`, `LAST_LOCATION`, `NUM_OBJECTS`, `NUM_LOCATIONS`, `NUM_CARRIED`, `NUM_WORN` |
| Textos | `DSTRINGS` |
| Fecha | `YEARHIGH`, `YEARLOW`, `MONTH`, `DAY` |
| Sonido | `PLAYSFX=1` … `FPLAYFLIL=10` |
| Ratón | `RESETMS` … `DELTAYMS`, valores 0 a 7 |
| Vocabulario | **Todo el vocabulario con el prefijo `_VOC_`** |
| **Versión** | **`V3` o `V2`** (`drf.pas:284-285`) |

El símbolo `V3` es el que permite a una misma fuente compilar en las 2 versiones:

```text
#ifdef "V3"
    SETAT 10 1
#else
    SET 200
#endif
```

`TEST.DSF` de TestUnitDAAD usa exactamente este mecanismo, y es la razón de que su DDB v3 sea
389 bytes mayor que el v2 mientras que `BLANK_EN.DSF`, que no lo usa, produzca 2 ficheros de
tamaño idéntico.

---

## 9. El formato JSON intermedio

El frontend no escribe bytes: emite un JSON que consume el backend. Documentarlo es útil porque
permite **sustituir cualquiera de las 2 mitades de forma independiente**.

Claves de primer nivel, verificadas sobre la salida real:

| Clave | Contenido |
|---|---|
| `settings` | Un objeto: `{classic_mode, debug_mode, v3code, maluva_used}` |
| `symbols` | Todos los símbolos definidos |
| `externs` | Binarios incorporados |
| `vocabulary` | `{VocWord, Value, VocType}` |
| `object_data` | `{Value, Noun, Adjective, Container, Wearable, Flags, Weight, InitialyAt}` |
| `connections` | `{FromLoc, ToLoc, Direction}` |
| `messages` | MTX: `{Value, Text}` |
| `sysmess` | STX: `{Value, Text}` |
| `locations` | LTX: `{Value, Text}` |
| `objects` | OTX: `{Value, Text}` |
| `xmessages` | Mensajes externos |
| `other_strings` | Cadenas auxiliares (por ejemplo el MML de `XPLAY`) |
| `processes` | `{Value, entries: [{Entry, Verb, Noun, condacts: […]}]}` |

Un condacto en el JSON:

```json
{"Opcode": 51, "Condact": "LET", "Indirection1": 0, "Indirection2": 1,
 "Param1": 100, "Param2": 200, "NumParams": 2}
```

`Indirection2` solo aparece cuando el frontend compiló con `-v3`. Es la señal que hace al backend
anteponer el condacto `INDIR`.

Nótese que **el JSON ya trae el `Opcode` numérico resuelto**, además del nombre. El backend puede
ignorar el nombre salvo para los pseudo-condactos, que llegan con códigos 128–143 y se traducen
allí.
