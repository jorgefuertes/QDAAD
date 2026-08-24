# 07 — Qué es exactamente DAAD v3

DAAD v3 es la primera versión nueva de DAAD desde los años noventa
(`daad-ready/WHATSNEW.TXT:4-5`). Este documento delimita con precisión qué cambia y, sobre todo,
qué **no** cambia.

---

## 1. La conclusión, primero

**v3 no es un formato nuevo. Es un modo de ejecución nuevo sobre el mismo formato.**

No hay ni un `#ifdef` de formato en el backend. La versión se selecciona con un único byte y se
despacha **en tiempo de ejecución** dentro del intérprete. La superficie en el compilador es
pequeña; la superficie semántica, en los intérpretes, es mayor.

### Prueba directa

**[V]** `BLANK_EN.DSF` compilado para ZX/128K con y sin `-v3` produce 2 DDB de **2038 bytes
cada uno** que difieren en **exactamente un byte**:

```text
offset 0x0000:  v2 = 02     v3 = 03
```

Nada más. Mismos punteros, mismas secciones, mismos tamaños.

### Qué NO es v3

Conviene decirlo explícitamente, porque son las suposiciones habituales:

| Suposición | Realidad |
|---|---|
| "v3 cambia la cabecera" | No. Los 34 bytes y los 13 vectores son idénticos |
| "v3 tiene superbancos o paginación" | No. Sigue siendo una imagen plana de ≤64 KB (`drb.php:2074`). La paginación existe solo en el fork ZXDAAD128, que es otra cosa |
| "v3 sube el límite de 255" | No. `MAX_MESSAGES_PER_TABLE`, `MAX_PROCESSES`, `MAX_OBJECTS`, `MAX_PARAMETER_RANGE` son incondicionales en `UConstants.pas:18-27` |
| "v3 amplía el espacio de opcodes" | No puede: el bit 7 del opcode está ocupado por la indirección. **Sí añade condactos, pero rellenando 3 huecos que ya existían** (§2) |
| "ZXDAAD128 es v3" | No. Escribe versión 3 pero es un formato bancarizado distinto. Ver [02-formato-ddb.md](02-formato-ddb.md#9-una-variante-que-no-es-daad-v3-zxdaad128) |

La forma real de superar los 255 mensajes no es v3 sino el modo no-clásico, que derrama MTX en
STX y LTX reescribiendo el opcode (`UMessageList.pas:76-109`).

---

## 2. Condactos nuevos, opcodes reutilizados

**v3 sí añade condactos.** `WHATSNEW.TXT:57` anuncia "New condact SETAT" y `:65` "New
metacondact GETKEY". Lo que no añade son **opcodes**: los 3 que estrena ocupaban ya huecos
llamados `dumb` en la tabla de v2, con cero parámetros y sin efecto.

`ApplyV3Changes` (`UCondacts.pas:276-292`), invocada desde `drf.pas:394` cuando se pasa `-v3`,
reescribe la tabla **en memoria**:

| Opcode | v2 | v3 | P | Tipos | ¿Lo escribe el autor? |
|---|---|---|---|---|---|
| 122 | `dumb` | **`INDIR`** | 1 | value | **No.** Lo genera el compilador ante un `@` en el segundo parámetro (§3) |
| 124 | `dumb` | **`SETAT`** | 2 | value, value | **Sí.** Es el condacto nuevo de v3 |
| 120 | `dumb` | **`XMES`** | 2 | — | **Sí**, pero ya existía en v2 como llamada a Maluva; v3 lo hace nativo (§6) |

Y fuera del binario, un pseudo-condacto nuevo: **`GETKEY`** (código 143), que el backend traduce
a `PAUSE 0` y que **exige `-v3`** (§7).

Resumiendo para un autor: v3 le da `SETAT`, `GETKEY`, `XMES` sin Maluva y la indirección del
segundo parámetro. Para un implementador de compilador, eso son 3 opcodes que antes no hacían
nada y un pseudo-condacto más.

`XMES` no aparece en `ApplyV3Changes` porque el frontend nunca lo genera directamente: el
pseudo-condacto `XMES` (código 128) lo traduce el backend a opcode 120 (`XMES_FINAL_OPCODE`,
`drb.php:45`).

---

## 3. Indirección del segundo parámetro

Es la novedad más visible para el autor y la más ingeniosa de implementación.

En v2 solo el primer parámetro admite `@`. Con `-v3`, `drf.pas:392` sube
`MAX_PARAM_ACCEPTING_INDIRECTION` de 1 a 2, y `LET 100 @200` pasa a ser sintaxis válida.

**[V]** Sin `-v3` el frontend rechaza el fuente:
`295:29:IND.DSF: Indirection is not allowed in this parameter.`

### Cómo funciona: código automodificable

No hay bit disponible en el opcode para señalarlo, así que el compilador **antepone un condacto
`INDIR`** (`drb.php:1139-1144`):

```php
if ($hasSecondParameterIndirection)
{
    writeByte($outputFileHandler, INDIR_OPCODE); // 122
    writeByte($outputFileHandler, $condact->Param2);
    $currentAddress+=2;
}
```

Y en tiempo de ejecución `INDIR` **parchea el DDB en RAM** (`PCDAAD/condacts.pas:2182-2189`):

```pascal
procedure _INDIR;
begin
 if (V3CODE) then writeByte(CondactPTR+3, getFlag(Parameter1));
 done := true;
end;
```

La aritmética: cuando `_INDIR` se ejecuta, `CondactPTR` ya apunta a su operando. Sea `P` esa
posición:

```text
P     = número de flag (operando de INDIR)
P+1   = opcode del condacto siguiente
P+2   = parámetro 1 del siguiente
P+3   = parámetro 2 del siguiente     ← CondactPTR+3, lo que se parchea
```

**[V]** `LET 100 @200` compilado para ZX/128K produce, en el offset 1715 del DDB:

```text
7A C8      INDIR 200          ; opcode 122, flag 200
33 64 C8   LET 100, 200       ; opcode 51, param1 = 100, param2 = marcador
```

El marcador del segundo parámetro es el propio número de flag; `INDIR` lo sobrescribe con el
contenido del flag antes de que `LET` lo lea.

### El marcador es indiferente

DRC escribe ahí el número de flag por comodidad —es el valor que ya tiene a mano
(`drb.php:1141`)—, pero **su contenido da igual**: el `INDIR` que precede lo sobrescribe siempre
antes de que nadie lo lea. El propio autor lo formula con un 0 (`DAAD V3 CAMBIOS.txt:13-14`):

```text
LET 100 @200  =>  INDIR 200   LET 100 0
```

Para un compilador nuevo esto significa que puede emitir lo que quiera en esa posición. Para un
intérprete, lo contrario y más importante: **no puede deducir nada del valor del marcador**, ni
usarlo para detectar que hubo indirección.

### Indirección doble

`@` en los dos parámetros a la vez es válido y **no necesita nada nuevo**: los dos mecanismos son
ortogonales. El primero va en el bit 7 del opcode y el segundo en el `INDIR` previo
(`DAAD V3 CAMBIOS.txt:14`):

```text
LET @100 @200  =>  INDIR 200   LET @100 0
```

En el backend se ve en el orden de las tres escrituras (`drb.php:1131` y `:1139-1144`): primero
se calcula `$opcode | 0x80` por la indirección del parámetro 1, después se emite el par
`INDIR, flagno`, y solo entonces el opcode ya marcado. El `INDIR` queda **delante del opcode con
el bit 7 puesto**, no entre el opcode y sus parámetros.

### Consecuencia crítica de portabilidad

**El DDB tiene que estar en memoria escribible.** Eso descarta ejecutar desde ROM y complica
cualquier intérprete que pagine el DDB en una ventana de solo lectura.

NextDAAD, cuyo DDB vive en páginas de 8 KB mapeadas, **no puede parchear**. Implementa `INDIR`
con un mecanismo de sustitución de un solo uso: guarda el valor en `indirArg2`/`indirValid` y lo
consume el recorredor de condactos al leer el siguiente parámetro 2
(`overlay0.asm:505-518`, `engine.asm:426-448`). El resultado observable es idéntico.

Cualquier intérprete nuevo debería seguir el camino de NextDAAD: es más limpio y no obliga a que
el DDB sea escribible.

---

## 4. `SETAT` y el banco alternativo de atributos

`SETAT valor, operación` manipula un bit del banco de atributos
(`PCDAAD/condacts.pas:2192-2210`):

```pascal
if (getFlagBit(FOBJECT_PRINT_FLAGS, 1)) then baseFlag := 91 else baseFlag := 59;
finalFlag := baseFlag - (Parameter1 div 8);
bit := Parameter1 mod 8;
Parameter2 := Parameter2 and 3;
if (Parameter2 = 3) then Parameter2 := 2;
case Parameter2 of
    0: ClearFlagBit(finalFlag, bit);
    1: SetFlagBit(finalFlag, bit);
    2: ToggleFlagBit(finalFlag, bit);
end;
```

Puntos a fijar:

- El direccionamiento es **descendente**: `flag = base − (valor div 8)`, `bit = valor mod 8`.
  Es exactamente el mismo esquema que ya usaban `HASAT` y `HASNAT`.
- La base es **59** normalmente, y **91** si el bit 1 del flag 53 está activo. Es decir, v3
  abre un segundo banco de atributos que ocupa los **flags 60 a 91**.
- La operación se enmascara con 3, y el valor 3 se trata como 2. Operaciones: **0 = borrar,
  1 = activar, 2 = conmutar**.

El uso previsto es dar atributos direccionables por bit a localidades y conexiones.
`MAX_BLOCKABLE_CONNECTIONS = 128` (`UConstants.pas:21`) es el techo pensado, aunque la constante
está declarada y nunca se usa.

> **Divergencia real.** `SETAT` está correctamente condicionado a v3 en PCDAAD, pero
> **`HASAT`/`HASNAT` aplican el banco alternativo sin comprobar la versión**
> (`PCDAAD/condacts.pas:1213-1226`). Un juego v2 que use el bit 1 del flag 53 como scratch verá
> sus `HASAT` leyendo los flags 60–91. msx2daad y NextDAAD sí lo condicionan; NextDAAD documenta
> la discrepancia y elige seguir a msx2daad (`overlay0.asm:436-441`).

---

## 5. Los 5 bits nuevos del flag 53

`msx2daad/include/daad.h:55-62`, confirmado en `NextDAAD/src/nextdaad.inc:555-563`.

| Bit | Valor | Constante | Significado |
|---|---|---|---|
| 0 | 1 | `F53_DOALLNONE` | El `DOALL` no encontró ningún objeto |
| 1 | 2 | `F53_ALTFLAGS` | Banco de atributos con base 91 en vez de 59 |
| 2 | 4 | `F53_NOPRONOUN` | Sin enclíticos para verbos de valor ≥ 240 |
| 4 | 16 | `F53_PREPFIRST` | Hubo una preposición antes del nombre 1 |
| 5 | 32 | `F53_UNRECWRD` | Hubo una palabra no reconocida tras el verbo |
| 6 | 64 | — | (v2) Listado continuo |
| 7 | 128 | `F53_LISTED` | (v2) `LISTOBJ` listó objetos |

Los bits 0, 4 y 5 los escribe el intérprete; los bits 1 y 2 los escribe el autor para configurar
el comportamiento. El bit 3 no se usa.

Estos 5 bits son, en la práctica, la mitad del valor de v3: permiten distinguir
"no entiendo la palabra X" de "no entiendo la frase", detectar preposiciones adelantadas y
reaccionar a un `DOALL` vacío.

---

## 6. `XMES` deja de depender de Maluva

En v2, imprimir un mensaje externo (un XMESSAGE, alojado en el fichero `.XMB`) requiere la
extensión Maluva y se codifica como una llamada `EXTERN` de 3 parámetros
(`drb.php:854-863`):

```text
EXTERN offsetLo, 3, offsetHi
```

En v3 es un condacto nativo de 2 parámetros (`drb.php:840-851`):

```text
opcode 120, offsetLo, offsetHi
```

> **El orden es siempre LSB, MSB, con independencia del endianness del target**
> (`DAAD V3 CAMBIOS.txt:40`). No es una excepción del formato sino una consecuencia de cómo se
> emite: los dos bytes no son un word, son dos parámetros de condacto sueltos, y `drb.php:849-850`
> los calcula sin mirar el target ni `$isLittleEndian`:
>
> ```php
> $condact->Param1 = $offset & 0xFF;          // Offset LSB
> $condact->Param2 = ($offset & 0xFF00) >> 8; // Offset MSB
> ```
>
> Es una trampa real para Amiga y ST, donde todo lo demás del DDB va en big endian. Lo mismo vale
> para la forma v2 vía `EXTERN`, donde el MSB viaja en `Param3` (`drb.php:860-862`).

El intérprete lee el segundo parámetro directamente en lugar de robar el byte siguiente
(`PCDAAD/condacts.pas:373-386`). El corolario está explícito en `PCDAAD/pcdaad.pas:291`:

```pascal
if V3CODE then MaluvaDisabled := true;
```

Es decir, **en v3 Maluva desaparece del camino**, con lo que se recuperan unos 2 KB de memoria
en las plataformas de 8 bits.

---

## 7. `PAUSE 0` y `GETKEY`

En v3, `PAUSE 0` deja de ser una pausa de 256/50 segundos y pasa a **esperar una pulsación,
dejando el código de tecla en los flags 60 y 61** —`fKey1` y `fKey2`, los mismos que usa `INKEY`
([05-flujo-ejecucion.md §6](05-flujo-ejecucion.md))— (`PCDAAD/condacts.pas:892`).

El pseudo-condacto `GETKEY` es azúcar sintáctico: el backend lo emite como `PAUSE 0`
(`drb.php:948-956`) y **exige `-v3`**:

```php
if ((!$v3code)) Error('GETKEY condact requires DAAD v3');
```

### Dos divergencias entre la especificación y los intérpretes

La especificación del autor dice que `PAUSE 0` espera a que se pulse una tecla **y a que se
vuelva a soltar** (`DAAD V3 CAMBIOS.txt:26`). **Ningún intérprete revisado espera la soltura:**

```pascal
procedure _GETKEY;                    { PCDAAD/condacts.pas:2063-2071 }
var inkey : word;
begin
  REPEAT UNTIL Keypressed;
  inkey := ReadKey;
  setflag(FKEY1, inkey and $FF);
  setflag(FKEY2, (inkey and $FF00) SHR 8);
  done := true;
end;
```

NextDAAD hace lo mismo con `key_wait_char` (`overlay0.asm:1877-1894`). La consecuencia práctica
es el rebote: dos `GETKEY` seguidos pueden devolver los dos la misma pulsación si el jugador
mantiene la tecla. Un intérprete nuevo debería seguir la especificación y esperar la soltura;
uno que quiera compatibilidad exacta con lo existente, no.

Y sobre el segundo flag: PCDAAD guarda en `fKey2` el **byte alto del código de tecla**, mientras
que NextDAAD lo pone **siempre a 0** (`overlay0.asm:1893`). Un juego portable no puede depender de
`fKey2` para distinguir teclas extendidas.

---

## 8. Cambios de comportamiento y deprecaciones

| Cambio | Detalle |
|---|---|
| `SYNONYM` deja de marcar `done` | Alinea el comportamiento con Amiga y Atari ST (`WHATSNEW.TXT:69`) |
| `XUNDONE` deprecado | Error de compilación con `-v3` (`drb.php:878`) |
| `XSPLITSCR` renativizado | v3: `GFX x 15`. v2: `EXTERN x 6` vía Maluva (`drb.php:995-1012`) |
| Plantilla española | `HABLA` pasa al valor 240 y el bit 2 del flag 53 va activo por defecto, resolviendo el bug de HABLA a costa de perder `HABLALO` (`WHATSNEW.TXT:71-72`) |

> **Nota sobre `SYNONYM`.** PCDAAD y jDAAD **nunca marcan `done`**, ni en v2 ni en v3
> (`condacts.pas:899-903`, `jdaad.js:2988-2992`), así que nunca implementaron correctamente el
> comportamiento v2. msx2daad y NextDAAD sí distinguen las dos versiones.

---

## 9. Constantes declaradas y nunca usadas

Tres pistas de v3 que quedaron sin implementar en el compilador:

| Constante | Fichero | Estado |
|---|---|---|
| `MAX_V3_DIRECTION = 127` | `UConstants.pas:20` | declarada, nunca referenciada |
| `MAX_BLOCKABLE_CONNECTIONS = 128` | `UConstants.pas:21` | declarada, nunca referenciada |
| `NUM_PREFIX_CONDACTS = 10` | `UConstants.pas:33` | declarada, nunca referenciada; parece un mecanismo de prefijos previsto y no construido |

Un compilador nuevo no debe implementarlas: no hay intérprete que las entienda.

---

## 10. Resumen ejecutable

Para generar v3, un compilador debe:

1. Escribir `3` en el byte `0x00`.
2. Admitir `@` en el segundo parámetro y, cuando aparezca, **anteponer `INDIR flagno`** al
   condacto, dejando en el parámetro 2 un marcador.
3. Reconocer `SETAT` como opcode 124 con 2 parámetros y `INDIR` como 122 con 1.
4. Emitir `XMES` como opcode 120 con offset bajo y alto, sin pasar por `EXTERN`.
5. Emitir `GETKEY` como `PAUSE 0` y rechazarlo si no se pidió v3.
6. Emitir `XSPLITSCR` como `GFX x 15`.
7. Rechazar `XUNDONE`.

Y **nada más**. Todo lo demás del binario es idéntico a v2.

Qué intérpretes lo soportan y en qué grado: [08-interpretes.md](08-interpretes.md).
