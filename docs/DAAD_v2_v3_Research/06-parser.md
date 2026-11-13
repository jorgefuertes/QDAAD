# 06 — El parser

Cómo DAAD convierte una línea escrita por el jugador en la *frase lógica* que consumen los
condactos: los flags 33 a 47.

Referencias principales: `PCDAAD/parser.pas` (modelo clásico) y
`msx2daad/src/daad_parser_sentences.c` (modelo por buffer).

> El fichero `PCDAAD/parser.pas` está codificado en CP437/CP850 y lleva un aviso de 12 líneas
> al principio. Abrirlo como UTF-8 corrompe las tablas de caracteres españoles.

---

## 1. La frase lógica

El parser no construye un árbol. Rellena 7 flags:

| Flag | Contenido |
|---|---|
| 33 | Verbo |
| 34 | Nombre 1 |
| 35 | Adjetivo 1 |
| 36 | Adverbio |
| 43 | Preposición |
| 44 | Nombre 2 |
| 45 | Adjetivo 2 |

Todos empiezan a `0xFF` (`NO_WORD`). Los condactos `ADJECT1`, `ADVERB`, `PREP`, `NOUN2` y
`ADJECT2` no son más que comparaciones contra estos flags; el verbo y el nombre 1 los compara
directamente el bucle de entradas (ver [05-flujo-ejecucion.md](05-flujo-ejecucion.md)).

Esto explica una limitación estructural de DAAD: **una frase solo puede tener 2 nombres y 2
adjetivos.** No hay forma de expresar "coge la llave, la espada y el mapa" en una sola frase
lógica; hay que trocearla.

---

## 2. Estructura del vocabulario

Descrita en [03-secciones.md](03-secciones.md#3-vocabulario): entradas de 7 bytes, palabra de
5 caracteres ofuscada con XOR `0xFF`, valor, tipo.

Tipos:

| 0 | 1 | 2 | 3 | 4 | 5 | 6 |
|---|---|---|---|---|---|---|
| VERB | ADVERB | NOUN | ADJECTIVE | PREPOSITION | CONJUNCTION | PRONOUN |

Dos intérpretes añaden tipos sintéticos que **no existen en el binario**, solo en su
representación interna:

- msx2daad: `UNKNOWN_WORD = 7`, para la palabra no reconocida de v3 (`daad.h:200-211`).
- ZXDAAD128: `PRONOUNVERB = 128` y `SEPARATOR = 129` (`DaadDefines.bas:135-136`).

### Rangos de valor con significado

| Rango | Constante | Efecto |
|---|---|---|
| < 14 | `MAX_DIRECTION_VOCABULARY` | Palabra de movimiento |
| ≤ 39 | `LAST_CONVERTIBLE_NOUN` | Nombre que puede convertirse en verbo |
| < 50 | `LAST_PROPER_NOUN` | Nombre propio: **no se memoriza como pronombre** |
| ≥ 240 | `LAST_PRONOMINAL_VERB` | Verbos excluidos de los enclíticos si el bit 2 del flag 53 está activo |

Estos rangos son **convenciones que el compilador no impone**. Un compilador nuevo debe
documentarlas al autor, no forzarlas.

---

## 3. Dos arquitecturas de parser

Los cinco intérpretes producen la misma frase lógica por dos caminos distintos.

### (A) Una pasada — PCDAAD, jDAAD, NextDAAD

1. **Al arrancar**, se cachean las conjunciones del vocabulario rodeadas de espacios (`' Y '`),
   hasta 256 (`parser.pas:118-143`).
2. Al leer la orden: se pasa a mayúsculas y **cada conjunción se sustituye por un punto**
   (`parser.pas:391-393`). Así, conjunción y separador acaban siendo lo mismo.
3. Separadores de frase: `.` `,` `;` `:`.
4. Se extrae **una sola orden** hasta el separador. El texto entre comillas se guarda aparte para
   `PARSE 1` (`parser.pas:446-455`).
5. Bucle palabra a palabra: se truncan a 5 caracteres, se busca en el vocabulario y **se asigna
   al primer hueco libre** mediante una cadena de `else if` (`parser.pas:519-531`).

El paso 5 es la clave de la semántica: el orden de las palabras importa poco; lo que importa es
el orden de los huecos. El primer nombre encontrado va a nombre 1, el segundo a nombre 2.

### (B) Tokenizar y poblar — msx2daad, ZXDAAD128

1. `parser()` recorre el texto y escribe pares `[id][tipo]` en `lsBuffer0`, o en `lsBuffer1` si
   está entre comillas (`daad_parser_sentences.c:34-143`).
2. `populateLogicalSentence()` consume `lsBuffer0` **hasta la primera conjunción** y rellena los
   flags (`:152-244`).
3. `nextLogicalSentence()` desplaza el buffer para la frase siguiente. Esto da **multi-frase
   real**, no una emulación con puntos.
4. `useLiteralSentence()` copia `lsBuffer1` a `lsBuffer0` para `PARSE 1`.

Tamaños: con `TEXT_BUFFER_LEN = 100`, `lsBuffer0` mide 51 bytes y `lsBuffer1` 26.

---

## 4. Post-proceso

Tras rellenar los huecos se aplican cuatro reglas (`parser.pas:577-604`).

### 4.1 Nombre convertible en verbo

Si no hay verbo y el nombre 1 está por debajo del umbral, **el nombre se copia al verbo sin
borrar el nombre**, de modo que ambos flags quedan con el mismo código:

```pascal
if (Option=0) and (getFlag(FVERB)=NO_WORD) and (getFlag(FNOUN)<=LAST_CONVERTIBLE_NOUN) then
  setFlag(FVERB, getFlag(FNOUN));
```

Es lo que hace que escribir sólo `NORTE` funcione.

Solo se aplica con `PARSE 0`, no con `PARSE 1`.

> **Divergencia de umbral.** PCDAAD, jDAAD, NextDAAD y ZXDAAD128 usan **40**. msx2daad usa **20**
> (`daad_parser_sentences.c:176`). NextDAAD resolvió la duda empíricamente contra el intérprete
> ZX original y dejó la transcripción en el comentario (`overlay1.asm:1370-1394`): el valor 39
> convierte y el 40 no. **msx2daad es la desviación**; un compilador debe documentar 40 y avisar
> de que los vocabularios que usen valores entre 20 y 39 se comportan distinto en MSX2.

### 4.2 Verbo heredado

Si falta verbo pero hay nombre, se reutiliza el verbo de la frase anterior. PCDAAD, jDAAD y
NextDAAD **solo lo hacen si la frase viene del buffer**, no si el jugador acaba de teclearla
(`parser.pas:586`). ZXDAAD128 lo hace siempre.

### 4.3 Pronombres

Se aplica el pronombre memorizado si no hay nombre. Y se memoriza el nombre actual como futuro
pronombre **solo si su valor es ≥ 50**: los nombres propios no se memorizan.

### 4.4 Resultado

`PARSE 0` devuelve cierto si quedó verbo o nombre. `PARSE n` con n > 0 siempre devuelve cierto.

---

## 5. Inglés frente a español

El bit 0 del byte `0x01` de la cabecera selecciona la familia de parser: activo para ES y PT,
inactivo para EN, DE y FR. No hay más idiomas en el binario.

### 5.1 Pronombres ingleses

En inglés el pronombre es una palabra independiente de tipo 6. Si no hay nombre, se copian los
flags 46 y 47 (`parser.pas:533-541`):

```pascal
else if (not IsSpanish) and (AWordRecord.AType = VOC_PRONOUN) and (not PronounInSentence) then
```

NextDAAD y msx2daad aplican la rama de pronombre **sin comprobar el idioma**, lo que en la
práctica es inocuo porque las bases españolas no declaran pronombres de tipo 6.

### 5.2 Enclíticos españoles

En español el pronombre va pegado al verbo: `CÓGELO`, `MÍRALA`, `DÁMELAS`. El parser detecta las
terminaciones `LO`, `LA`, `LOS`, `LAS` sobre la palabra **completa** —no sobre los 5 caracteres
truncados— y, si la encuentra, se comporta como si hubiera aparecido un pronombre.

Aquí es donde más divergen las cinco implementaciones, y el motivo tiene nombre propio:

> **El bug de HABLA.** El verbo `HABLAR` en imperativo es `HABLA`, que **termina en `LA`**. Un
> parser ingenuo lo interpreta como "habla" + pronombre "la", y el jugador acaba hablando con
> algo que no ha nombrado.

| Intérprete | Cómo lo resuelve |
|---|---|
| **PCDAAD** | Detecta la terminación y **no reverifica**. Sufre el bug (`parser.pas:544-566`) |
| **jDAAD** | Quita el sufijo, trunca a 5 y **vuelve a buscar en el vocabulario como verbo**. Solo aplica el pronombre si el resto sigue siendo verbo. El comentario cita el bug de HABLA explícitamente (`jdaad.js:1484-1519`) |
| **msx2daad** | Inyecta un token sintético `[SYNTH_PRONOUN_ID, PRONOUN]` en `lsBuffer`. Respeta el bit `F53_NOPRONOUN` para verbos con valor > 239 |
| **ZXDAAD128** | Reverifica exigiendo que el identificador y el tipo coincidan, y marca el token como `PRONOUNVERB` |
| **NextDAAD** | **No los implementa** (documentado en `manual/known-differences.md`) |

La solución de fondo llegó con DAAD v3: el **bit 2 del flag 53** desactiva los enclíticos para
los verbos de valor ≥ 240, y la plantilla española de DAAD Ready mueve `HABLA` al valor 240 con
ese bit activo por defecto (`daad-ready/WHATSNEW.TXT:71-72`). El precio es perder `HABLALO`.

> **Bug en jDAAD.** `limitEnclicitPronouns()` comprueba el **bit 6** del flag 53
> (`jdaad.js:626-629`) donde debería comprobar el bit 2. En jDAAD el bit 6 es el de listado
> continuo, así que activar el listado continuo desactiva la limitación de enclíticos y
> viceversa. Ver [13-portabilidad.md](13-portabilidad.md).

### 5.3 Artículos y nombres de objeto

En los textos, `_` se sustituye por el nombre del objeto y `@` por el nombre con artículo
capitalizado. **`@` solo funciona en bases españolas** (`jdaad.js:1646`).

Cómo se eliminan los artículos del nombre almacenado:

- jDAAD `replaceArticles()` **borra la primera palabra, sea cual sea** (`jdaad.js:1568`).
- NextDAAD borra únicamente `a`, `an`, `some` y `the`, y documenta la diferencia como decisión
  deliberada y trampa conocida de portado.

NextDAAD y ZXDAAD128 truncan además el nombre del objeto en el primer punto.

### 5.4 Caracteres españoles

`fixSpanishCharacters` (`parser.pas:327-350`) mapea los caracteres de entrada al juego interno de
DAAD. jDAAD replica la tabla con la cadena `'º¡¿«»áéíóúñÑçÇüÜ'` y el desplazamiento `16 +
indexOf` (`jdaad.js:1183-1212`), que es exactamente la codificación 16–31 descrita en
[03-secciones.md](03-secciones.md#23-codificación-de-caracteres).

---

## 6. Multi-frase

Una orden como `COGE LA LLAVE Y ABRE LA PUERTA` produce dos frases lógicas. El intérprete
procesa la primera; el DDB debe volver a llamar a `PARSE 0` para obtener la siguiente. El
condacto `NEWTEXT` descarta lo que quede pendiente.

- En el modelo (A) la conjunción se convierte en separador y el resto de la línea queda en un
  buffer.
- En el modelo (B) el buffer de tokens es explícito y `nextLogicalSentence()` avanza sobre él.

Consecuencia práctica: `NEWTEXT` es imprescindible en las respuestas que cambian el estado de
forma incompatible con el resto de la orden, y su omisión produce comportamientos distintos
según el modelo.

---

## 7. Historial de órdenes

Solo PCDAAD (10 órdenes, `parser.pas:610-638`) y NextDAAD (`inp_recall_last`) implementan
recuperación de órdenes anteriores. No forma parte del formato.

---

## 8. Novedades de v3 en el parser

2 bits nuevos del flag 53, ambos escritos por el parser:

| Bit | Constante | Cuándo se activa |
|---|---|---|
| 4 (16) | `F53_PREPFIRST` | Apareció una preposición **antes** del nombre 1 (`parser.pas:528`) |
| 5 (32) | `F53_UNRECWRD` | Apareció una palabra **no reconocida** después del verbo (`parser.pas:572`) |

El segundo es el que permite por fin responder "no conozco la palabra X" en lugar de un genérico
"no te entiendo". Detalles en [07-daad-v3.md](07-daad-v3.md).
