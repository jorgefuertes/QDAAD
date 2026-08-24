# 15 — Límites de DAAD

Todos los rangos y topes del formato, con la distinción que más importa a quien escribe un
compilador: **lo que el byte permite** frente a **lo que se puede usar de verdad**.

La tabla estructural vive en
[02-formato-ddb.md §7](02-formato-ddb.md#7-límites-del-formato). Este documento la desarrolla,
resuelve las contradicciones entre las constantes de DRC y lo que el backend escribe realmente, y
señala los tres puntos donde el compilador de referencia deja pasar una base de datos corrupta.

Nada de esto cambia en v3 (`UConstants.pas:18-27`, [07-daad-v3.md](07-daad-v3.md#1-la-conclusión-primero)).

---

## 1. Resumen

| Entidad | Rango técnico | **Rango usable** | Reservados | ¿Lo comprueba DRC? |
|---|---|---|---|---|
| Localidad | 0–255 | **0–251** | 252–255 son centinelas de ubicación | **No** |
| Palabra (valor) | 0–255 | **0–254** | 255 = `NO_WORD`, comodín `_` | No aplica |
| Objeto | 0–255 | **0–254** | ninguno | Sí, pero mal (§4) |
| Proceso | 0–255 | 0–254 | 0, 1 y 2 son de arranque | **No** |
| Mensaje (por tabla) | 0–255 | 0–254 | los STX bajos son de sistema | Sí |
| Flag | 0–255 | 0–255 | 0–63 son de sistema | No aplica |
| Valor de un flag | 0–255 | 0–255 | — | No aplica |

La razón de fondo es siempre la misma: **los contadores de la cabecera son de 1 byte**
(`drb.php:1849-1862`, [02-formato-ddb.md §2](02-formato-ddb.md)), y el backend los escribe sin
comprobar el desbordamiento:

```php
function writeByte($handle, $byte)
{
    fputs($handle, chr($byte), 1);   // drb.php:60-63
}
```

`chr()` en PHP toma el argumento módulo 256. Cualquier contador que llegue a 256 se escribe como
**0**. De ahí el «rango usable» de la tabla: el número de entidades nunca puede pasar de 255, lo
que deja los identificadores en 0–254.

---

## 2. Localidades

### Rango

Contador en el byte `0x04` de la cabecera (`drb.php:1852-1853`). Se numeran **desde 0 y de forma
consecutiva**, y todas deben aparecer en `/CON` aunque no tengan salidas
(`USintactic.pas:425`, [03-secciones.md §5](03-secciones.md)).

El tope real **no son 255 sino 252**, porque el bloque alto está ocupado por los centinelas de
ubicación de objeto (`UConstants.pas:9-13`):

| Valor | Constante | Significado |
|---|---|---|
| 252 | `LOC_NOT_CREATED` | El objeto no existe todavía |
| 253 | `LOC_WORN` | Puesto por el jugador |
| 254 | `LOC_CARRIED` | Llevado por el jugador |
| 255 | `LOC_HERE` | En la localidad actual |

Lo confirma el análisis semántico de DRC por partida doble:

- Para el tipo de parámetro `locno_`, un valor fuera de rango solo se acepta **si es ≥ 252**
  (`UCondacts.pas:259`). Para `locno` a secas, cualquier valor `>= LTXCount` es error
  (`UCondacts.pas:252`).
- Al asignar la ubicación inicial de un objeto se admite `< NUM_LOCATIONS`, 252, 253 o 254 —
  pero **no 255** (`USintactic.pas:461`). `HERE` no tiene sentido antes de empezar la partida.

En ejecución, mandar un objeto a la localidad 255 es el error 2
([05-flujo-ejecucion.md §7](05-flujo-ejecucion.md)).

> Estrictamente, una aventura puede **declarar** hasta 255 localidades y la 252 sería un destino
> válido para el jugador. Pero ningún objeto podría estar nunca allí, así que el techo práctico
> para un juego normal es **252 localidades, numeradas 0–251**.

### Localidades reservadas por contenedores

Un objeto dentro de un contenedor se codifica como «estar en la localidad de número igual al
identificador del contenedor» ([03-secciones.md §4.2](03-secciones.md)). DRC avisa si un objeto
contenedor no tiene reservada esa localidad (`drb.php:744-749`).

Consecuencia para un compilador: **el número de un objeto contenedor está acotado por el número
de localidades**, no solo por el de objetos.

### La 0 no es especial

Ningún fuente le da semántica. Las únicas menciones son estructurales: es la primera entrada de
la tabla de conexiones ([14-verificacion.md](14-verificacion.md)) y su lista termina en `0xFF`
como todas. No existe el concepto de *limbo*: un objeto no creado se marca con 252, no
mandándolo a una localidad basura.

---

## 3. Palabras de vocabulario

### El valor

Es de 1 byte, así que 0–255, y el 255 es `NO_WORD` (`UConstants.pas:13`), que en el fuente `.DSF`
se escribe `_` y funciona como comodín en las entradas de proceso
([03-secciones.md §6.2](03-secciones.md)) y en los pares nombre/adjetivo de objeto. Quedan
**0–254**.

### El número de entradas no tiene tope

La tabla de vocabulario **no lleva contador**: son entradas de 7 bytes terminadas por un único
byte `0x00` ([03-secciones.md §3](03-secciones.md), `drb.php:698-709`). El único límite es la
imagen de 64 KB.

Y varias entradas pueden compartir valor: así es como se codifican los sinónimos. El vocabulario
tiene, por tanto, **dos cardinalidades distintas** —número de entradas y número de valores
distintos— y solo la segunda está acotada.

### Longitud

**5 caracteres** (`VOCABULARY_LENGTH`, `UConstants.pas:15`). Las palabras se truncan, no se
rechazan (`parser.pas`, [06-parser.md §3](06-parser.md)). En el DDB van en mayúsculas, rellenadas
con espacios hasta 5 y con cada byte en XOR `0xFF`.

### Subrangos por convención

**No los impone el compilador** ([06-parser.md §2](06-parser.md): «un compilador nuevo debe
documentarlas al autor, no forzarlas»):

| Rango | Constante | Efecto |
|---|---|---|
| 0–13 | `MAX_DIRECTION_VOCABULARY` | Palabra de movimiento; es lo que va en `/CON` |
| ≤ 39 | `MAX_CONVERTIBLE_NAME` | Nombre convertible en verbo si la frase no trae ninguno |
| < 50 | `LAST_PROPER_NOUN` | Nombre propio: no se memoriza como pronombre |
| ≥ 240 | `LAST_PRONOMINAL_VERB` | Verbos excluidos de enclíticos con el bit 2 del flag 53 |

Trampa de portabilidad: **msx2daad usa umbral 20 en vez de 40** para el convertible
(`daad_parser_sentences.c:176`). Los valores 20–39 se comportan distinto en MSX2
([13-portabilidad.md](13-portabilidad.md)).

---

## 4. Objetos

### El número, y la contradicción de DRC

`UConstants.pas:22` declara `MAX_OBJECTS = 256` y el frontend valida con:

```pascal
if (OTXCount > MAX_OBJECTS) THEN SyntaxError('Too many objects, maximum allowed is ' + ...);
                                                          // USintactic.pas:363
```

Es decir, **acepta 256 objetos**. Pero el backend escribe el contador en un byte
(`drb.php:1849-1850`), y con 256 objetos la cabecera acaba declarando **0**.

> **Bug de DRC.** La comprobación debería ser `>= MAX_OBJECTS`, o la constante valer 255. Tal
> como está, una aventura con exactamente 256 objetos compila sin un solo aviso y produce un DDB
> con la cuenta de objetos a cero. El límite efectivo es **255 objetos, numerados 0–254**, y un
> compilador nuevo debe rechazar el 256.

### Sin números reservados

No hay ningún objeto con significado especial. Que el objeto 0 sea la linterna es **convención de
la plantilla**, visible en el `BLANK` (`12-formato-dsf.md:55`), no una regla del formato.

### Rangos internos del objeto

| Campo | Rango | Fuente |
|---|---|---|
| Peso | 0–63 (6 bits) | `MAX_WEIGHT`, `UConstants.pas:25` |
| Contenedor | bit 6 del byte de peso | [03-secciones.md §4.2](03-secciones.md) |
| Vestible | bit 7 del byte de peso | ídem |
| Nombre, adjetivo | valor de vocabulario, `0xFF` = ninguno | [03-secciones.md §4.1](03-secciones.md) |
| Ubicación inicial | `< NUM_LOCATIONS`, o 252/253/254 | `USintactic.pas:461` |

Los datos de objeto **no forman un registro**: son 4 arrays paralelos con su propio puntero de
cabecera cada uno ([03-secciones.md §4](03-secciones.md)).

---

## 5. El resto de los topes

| Límite | Valor | Constante / fuente |
|---|---|---|
| Procesos | 255 | `MAX_PROCESSES`, `UConstants.pas:18` |
| Mensajes por tabla (MTX/STX/LTX/OTX) | 255 | `MAX_MESSAGES_PER_TABLE`, `UConstants.pas:24` |
| Mensajes totales en modo no clásico | 3 × 255 | `USintactic.pas:659-660` |
| Opcodes de condacto | 128 | el bit 7 señala indirección |
| Parámetros por condacto | 3 | `MAX_CONDACT_PARAMS`, `UConstants.pas:19` |
| Rango de un parámetro | 0–255 | `MAX_PARAMETER_RANGE`, `UConstants.pas:27` |
| Valor de un flag | 0–255 | `MAX_FLAG_VALUE`, `UConstants.pas:14` |
| Etiquetas | 1024 | `MAX_LABELS`, `UConstants.pas:29` |
| Salto relativo | ±128 entradas, sin salir del proceso | `USintactic.pas:76` |
| Tamaño total del DDB | 65535 − dirección base | `drb.php:2074` |
| XMessages | 64 KB en total, y por target | `drb.php:420-445`, [11-build-plataformas.md §3.3](11-build-plataformas.md) |

Dos constantes están declaradas y **nunca se usan**: `MAX_V3_DIRECTION = 127` y
`MAX_BLOCKABLE_CONNECTIONS = 128` ([07-daad-v3.md §9](07-daad-v3.md)). No hay que implementarlas.

---

## 6. Qué debe comprobar un compilador nuevo

DRC solo valida el número de **objetos** (mal, §4), el de **mensajes** y el de **etiquetas**. El
resto se desborda en silencio. La lista mínima:

1. **Localidades ≤ 252.** Ni DRC ni ningún intérprete lo comprueban. Por encima de 251 hay
   localidades donde ningún objeto puede estar; a partir de 256 la cabecera miente.
2. **Objetos ≤ 255.** Rechazar el 256 que DRC acepta.
3. **Procesos ≤ 255.** Sin comprobación en DRC.
4. **Un objeto contenedor exige que exista la localidad de su mismo número.**
5. **Valor de palabra ≠ 255**, reservado para `NO_WORD`.
6. **Avisar, no rechazar,** cuando un valor de vocabulario caiga en los subrangos convencionales
   de §3, y avisar del rango 20–39 por msx2daad.
7. **La ubicación inicial de un objeto no puede ser 255** (`HERE`).
8. **Peso ≤ 63.**
9. **Dirección final ≤ 0xFFFF**, que es el tope duro de todo.

---

## 7. Diferencias entre versiones y plataformas

**No hay ninguna.** Todos los límites son incondicionales en `UConstants.pas:18-27`: no cambian
entre v2 y v3, ni entre `BIT8` y `BIT16`, ni entre targets. Lo que sí cambia por plataforma es la
dirección base, el endianness y la alineación ([02-formato-ddb.md](02-formato-ddb.md)), no las
cuentas.

La única forma real de superar los 255 mensajes es el **modo no clásico**, que derrama MTX en STX
y LTX reescribiendo el opcode (`UMessageList.pas:76-109`). Ese truco sirve **solo para mensajes**:
nunca para localidades ni para objetos.

`ZXDAAD128` sí tiene límites distintos, pero **no es DAAD v3**: es un formato bancarizado propio
con cabecera de 58 bytes ([02-formato-ddb.md §9](02-formato-ddb.md)).

---

## 8. Por qué el último ID es el 254 y no el 255

La objeción es razonable: 8 bits dan 256 valores, del 0 al 255. Y es cierto que **el campo del
identificador puede almacenar el 255**; PCDAAD, sin ir más lejos, dimensiona sus arrays a 256 y lo
manejaría sin inmutarse:

```pascal
const NUM_OBJECTS = 256;              { objects.pas:11-12 — constantes fijas,  }
      MAX_OBJECT  = NUM_OBJECTS - 1;  { no leídas de la cabecera              }
var objLocations: array [0..NUM_OBJECTS-1] of byte;
```

Lo que no cabe en un byte no es el identificador, es **la cuenta**. Con N objetos numerados
0…N−1, tener uno con ID 255 exige N = 256, y el contador de la cabecera se escribe con
`chr(256)` = **0**.

Y ese 0 no se reinterpreta como 256 en ninguna parte. En PCDAAD, `numObj` es un `byte`
(`ddb.pas:16`) y gobierna todos los recorridos:

```pascal
for i := 0 to DDBHeader.numObj - 1 do                    { objects.pas:150 }
 setObjectLocation(i, getByte(DDBHeader.objInitiallyAtPos + i));
```

Con `numObj` a 0 la expresión vale −1 y **el bucle no se ejecuta ni una vez**: ningún objeto
recibe su localización inicial y todos se quedan en `NOT_CREATED`. `getObjectFullWeight` deja
fuera a todos por la misma razón (`objects.pas:98`). No es un desbordamiento benigno: la aventura
se queda sin objetos.

Conviene tener presente que **los tres límites tienen causas distintas**, y que ninguna es el
ancho del campo:

| Entidad | Qué recorta el rango |
|---|---|
| Objeto | El **contador** de la cabecera, que no puede decir 256 |
| Palabra | El **centinela** `NO_WORD` = 255. No hay contador: la tabla termina en `0x00` |
| Localidad | **Cuatro centinelas**, 252–255, que ninguna localidad real puede usar |

De ahí que el vocabulario, que no tiene contador, pierda igualmente el 255, y que las localidades
pierdan cuatro valores en vez de uno.
