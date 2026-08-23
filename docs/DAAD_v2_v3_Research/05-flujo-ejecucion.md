# 05 — Flujo de ejecución

Cómo se ejecuta un DDB: el bucle del intérprete, la pila de procesos, `DOALL`, el indicador
`done` y la tabla completa de flags de sistema.

---

## 1. El malentendido más común

**El intérprete no tiene un bucle de juego.** No hay nada cableado del tipo "describe la
localidad, lee la orden, recorre la tabla de respuestas, procesa el timeout". Todo eso lo
escribe el autor en el DDB.

Lo único que hace el intérprete al arrancar es **apilar el proceso 0 y empezar a ejecutarlo**.
El bucle de juego es literalmente el proceso 0, que por convención de las plantillas llama a
otros procesos:

```text
PROCESO 0   →  bucle principal escrito por el autor
  ├─ PICTURE / DISPLAY   dibuja la localidad
  ├─ DESC                describe la localidad
  ├─ PROCESS 3           equivalente al proceso 1 de PAWS
  ├─ PROCESS 1           tabla de respuestas
  └─ …
```

Cuando la pila de procesos se vacía, el comportamiento varía: NextDAAD **reinicia el proceso 0**
(`engine.asm:271-275`, comentado como "the game never ends"), mientras que PCDAAD hace `StackPop`
y continúa donde estaba. Un compilador no debe apoyarse en ese detalle.

Los procesos 0, 1, 2 y 6 son internos de la plantilla; el 3 se corresponde con el proceso 1 de
PAW, el 4 con el 2, y el 5 con la tabla de respuestas.

---

## 2. El bucle, paso a paso

La implementación de referencia es `PCDAAD/pcdaad.pas:31-165`, escrita con `goto` y 2
etiquetas, `RunEntry` y `RunCondact`. Los 5 intérpretes reproducen esta máquina.

### `RunEntry` — seleccionar una entrada

1. Si el byte del verbo vale `0x00`, se ha llegado al **fin de la tabla de entradas**:
   - Si hay un `DOALL` activo, busca el siguiente objeto en la localidad del `DOALL`. Si lo
     encuentra, fija el objeto referenciado, actualiza el flag 50 y **salta de vuelta a la
     entrada y condacto donde estaba el `DOALL`** (`pcdaad.pas:69-71`).
   - Si no hay más objetos, desactiva el `DOALL` y continúa.
   - Después, desapila el proceso, **avanza un byte el puntero de condactos** y salta a
     `RunCondact` (`pcdaad.pas:87-89`). Es decir: al volver de un `PROCESS` se reanuda en el
     condacto siguiente al `PROCESS`.
2. La entrada casa si **verbo y nombre coinciden con los flags 33 y 34, o valen `0xFF`**
   (comodín):
   ```pascal
   ValidEntry := ((getByte(EntryPTR) = getFlag(FVERB)) OR (getByte(EntryPTR) = NO_WORD))
                 and ((getByte(EntryPTR+1) = getFlag(FNOUN)) OR (getByte(EntryPTR+1) = NO_WORD));
   ```
3. Se lee el puntero al bloque de condactos del word en `EntryPTR+2`.
4. Si no casa, `EntryPTR += 4` y se repite.

### `RunCondact` — ejecutar un condacto

1. Lee el opcode. Si vale `0xFF`, fin del bloque: `EntryPTR += 4` y vuelve a `RunEntry`.
2. **Reescribe los flags 62 y 29 en cada iteración** (`pcdaad.pas:117-120`), para que un juego no
   pueda falsear la detección de plataforma. Ver §5.
3. Si el bit 7 del opcode está activo: marca indirección y lo enmascara con `0x7F`.
4. Lee 0, 1 o 2 bytes de parámetro según la aridad del opcode. **Con indirección, el primer
   parámetro se sustituye por el contenido del flag que designa** (`pcdaad.pas:141`).
5. Ejecuta el condacto.
6. Si devolvió falso (una condición que no se cumple), `EntryPTR += 4` y vuelve a `RunEntry`.
   Si devolvió cierto, avanza al condacto siguiente.

### El caso especial de jDAAD

JavaScript no puede bloquear el hilo, así que `run()` **retorna** cuando entra en un estado que
requiere esperar al usuario —`inPARSE`, `inANYKEY`, `inQUIT`, `inEND`, `inSAVE`, `inLOAD`,
`inINKEY`— y el manejador de teclado la vuelve a invocar (`jdaad.js:1123`, `1300-1318`). El
código lleva un comentario de 12 líneas explicando por qué no puede parecerse al Pascal
original. Es un detalle de implementación, no un cambio de semántica.

---

## 3. La pila de procesos

| Intérprete | Profundidad | Estructura |
|---|---|---|
| PCDAAD | ilimitada (`stack.pas`) | registro completo |
| jDAAD | ilimitada (array JS) | `processPTR`, `entryPTR`, `condactPTR`, `doallPTR`, `doallEntryPTR`, `doallFlag`, `doallLocation`, `currentProcess` |
| msx2daad | **10** (`NUM_PROCS`) → error 3 | 8 campos, con `DOALL` por nivel |
| NextDAAD | `PROC_DEPTH` → error 3 | **5 bytes por nivel**: `[num][entryPtr][condactPtr]` |
| ZXDAAD128 | **10** | 9 arrays paralelas |

En NextDAAD, `condactPtr == 0` significa "estoy en la cabecera de la entrada", lo que le permite
codificar todo el estado en 5 bytes.

### Divergencia real: qué pasa con `done` al hacer `PROCESS`

- PCDAAD, jDAAD, msx2daad y ZXDAAD128 **ponen `done` a falso** al apilar un proceso
  (`PCDAAD/condacts.pas:1375`, `msx2daad/src/daad_condacts.c:126`).
- NextDAAD **no toca `done` al desapilar**, deliberadamente (`engine.asm:344-346`), para que
  `PROCESS n` seguido de `ISDONE` informe del resultado del subproceso.

Un juego que dependa de `ISDONE` inmediatamente después de un `PROCESS` se comporta distinto en
NextDAAD que en el resto.

---

## 4. `DOALL`

`DOALL loc` recorre todos los objetos de una localidad ejecutando el resto de la entrada una vez
por objeto. La mecánica está en `RunEntry`: al llegar al fin de la tabla de entradas, si hay un
`DOALL` activo se vuelve atrás.

Puntos donde los intérpretes divergen:

| Aspecto | Comportamiento |
|---|---|
| Ámbito | **Por nivel de proceso** en msx2daad y ZXDAAD128; **global** en NextDAAD y PCDAAD |
| Supervivencia a `PROCESS` | Sí, salvo en NextDAAD: su flag 50 es global y no sobrevive (documentado en `NextDAAD/manual/known-differences.md`) |
| `DOALL` anidado | PCDAAD: "Runtime error 4" + `EXIT 0`. NextDAAD: error 4 ruidoso, explícitamente por diseño. jDAAD: `NESTED_DOALL_ENABLED = false` |
| Cláusula `EXCEPT` | Se implementa comparando nombre y adjetivo del objeto con el segundo nombre de la frase (`pcdaad.pas:60-67`) |
| Sin objetos (v3) | Activa el bit 0 del flag 53 (`F53_DOALLNONE`) y dispara `NOTDONE` |

El arreglo del comportamiento con cero objetos se aplicó de forma coordinada en ADP, PCDAAD,
jDAAD y msx2daad (`daad-ready/WHATSNEW.TXT:39-41`).

---

## 5. Flags 29 y 62: detección de plataforma

Los flags 29 (`fGFlags`) y 62 (`fScMode`) permiten a un juego saber sobre qué está corriendo.
PCDAAD los **reescribe después de cada condacto** para que no puedan falsearse.

El problema es que cada intérprete publica valores distintos:

| Intérprete | Flag 29 | Flag 62 | Fuente |
|---|---|---|---|
| PCDAAD | `129` (`128+1`) | `141` (`13+128`) | `pcdaad.pas:119-120` |
| jDAAD | `129` | `142` (`14+128`) | `jdaad.js:1086-1087` |
| msx2daad | `128` — **sin el bit de ratón** | `16 \| SCREEN` (24 para SC8) | `daad_platform_msx2.c:585-590` |
| NextDAAD | `129` | `144` | `manual/platform-notes.md` |
| **ZXDAAD128** | **nunca se escribe (0)** | **nunca se escribe (0)** | — |

El caso de ZXDAAD128 es un fallo con consecuencias visibles: `HASAT GMODE` comprueba el bit 7
del flag 29, que allí vale siempre cero. **Un juego que condicione el dibujado de imágenes a
`HASAT GMODE` no dibuja nada en ZXDAAD128.**

Las máscaras estándar (`msx2daad/include/daad.h:44-64`):

| Constante | Valor para `HASAT` | Significado |
|---|---|---|
| `HAS_MOUSE` | 240 | flag 29, bit 0 — ratón disponible |
| `HAS_GMODE` | 247 | flag 29, bit 7 — gráficos disponibles |
| `HAS_CONTAINER` | 31 | flag 56, bit 7 |
| `HAS_WAREABLE` | 23 | flag 57, bit 7 |
| `HAS_LISTED` | 55 | flag 53, bit 7 |
| `HAS_TIMEOUT` | 87 | flag 49, bit 7 |

El direccionamiento de `HASAT`/`HASNAT`/`SETAT` es **descendente desde el flag 59**:
`flag = 59 − (valor div 8)`, `bit = valor mod 8`. En DAAD v3 la base puede pasar a 91; ver
[07-daad-v3.md](07-daad-v3.md).

---

## 6. Tabla de flags de sistema

Referencias: `msx2daad/include/daad.h:215-257`, `PCDAAD/flags.pas:9-46`,
`ZXDAAD128/src/DaadDefines.bas:15-56`, `jdaad.js:151-186`.

| Flag | Nombre | Significado |
|---|---|---|
| 0 | `fDark` | Distinto de cero: oscuridad |
| 1 | `fNOCarr` | Número de objetos llevados (no vestidos) |
| 2–3 | `fWork1`, `fWork2` / `fFULL` | Trabajo interno del sistema |
| 20 | `fMALUVA` | Resultado y errores de Maluva. **Solo real en ZXDAAD128**; PCDAAD lo declara con el comentario "not really used" |
| 21 | `FSOUND` | **Solo PCDAAD**: control del motor de sonido. bit 0 SFX activo, bit 1 OPL activo, bit 2 SFX en bucle, bit 3 OPL en bucle, bit 4 SFX sonando, bit 5 OPL sonando |
| 23 | `fEMPTY` | Base de la pila de DAAD |
| 24 | `fStack` | Pila pequeña: 10 apilamientos de 2 bytes |
| 25 / 26 / 27 | `fO2Num` / `fO2Con` / `fO2Loc` | Objeto 2, resuelto desde nombre2 y adjetivo2 |
| 28 | `fDarkF` | Lo usa la plantilla, no el intérprete |
| 29 | `fGFlags` | bit 7 gráficos, bit 0 ratón. Ver §5 |
| 30 | `fScore` | Puntuación (convención, opcional) |
| 31–32 | `fTurns` | Número de turnos, 2 bytes little-endian |
| 33 | `fVerb` | Verbo de la frase lógica actual |
| 34 | `fNoun1` | Primer nombre |
| 35 | `fAdject1` | Adjetivo del primer nombre |
| 36 | `fAdverb` | Adverbio |
| 37 | `fMaxCarr` | Máximo de objetos transportables (`ABILITY`), inicialmente 4 |
| 38 | `fPlayer` | Localidad actual del jugador |
| 39–40 | `fO2Att` | Atributos del objeto 2 |
| 41 | `fInStream` | Stream de entrada; se usa módulo 8 |
| 42 | `fPrompt` | Mensaje de sistema usado como prompt; 0 elige uno de 4 al azar |
| 43 | `fPrep` | Preposición |
| 44 | `fNoun2` | Segundo nombre |
| 45 | `fAdject2` | Adjetivo del segundo nombre |
| 46–47 | `fCPNoun`, `fCPAdject` | Pronombre actual ("ello") |
| 48 | `fTime` | Duración del timeout |
| 49 | `fTIFlags` | Bitmask de control del timeout (ver abajo) |
| 50 | `fDAObjNo` | Objeto actual del bucle `DOALL` |
| 51 | `fCONum` | Último objeto referenciado por `GET`/`DROP`/`WEAR`/`WHATO`… |
| 52 | `fStrength` | Fuerza del jugador: peso máximo que puede llevar, inicialmente 10 |
| 53 | `fOFlags` | Indicadores de impresión de objetos, `DOALL` y parser. bit 7 = hubo objetos listados. **En v3 se le añaden 5 bits**; ver [07-daad-v3.md](07-daad-v3.md) |
| 54 | `fCOLoc` | Localidad del objeto referenciado |
| 55 | `fCOWei` | Peso del objeto referenciado |
| 56 | `fCOCon` | 128 si el objeto referenciado es contenedor |
| 57 | `fCOWR` | 128 si el objeto referenciado es vestible |
| 58–59 | `fCOAtt` | Atributos de usuario del objeto referenciado |
| 60–61 | `fKey1`, `fKey2` | Código de la tecla devuelta por `INKEY` y, en v3, por `PAUSE 0` |
| 62 | `fScMode` | Modo de pantalla. Ver §5 |
| 63 | `fCurWin` | Ventana activa. Solo lectura |
| **64–254** | — | **Libres para el autor.** La convención es empezar por el 254 y bajar |
| 60–91 | — | En v3, banco alternativo de atributos si el bit 1 del flag 53 está activo |

### Bitmask del flag 49 (`fTIFlags`)

`msx2daad/include/daad.h:71-79`:

| Bit | Valor | Significado |
|---|---|---|
| 0 | 1 | El timeout solo puede ocurrir en el primer carácter (`TIME`) |
| 1 | 2 | El timeout puede ocurrir en el "More…" (`TIME`) |
| 2 | 4 | El timeout puede ocurrir en `ANYKEY` (`TIME`) |
| 3 | 8 | Limpiar la ventana tras la entrada (`INPUT`) |
| 4 | 16 | Imprimir la entrada en el stream actual tras editar (`INPUT`) |
| 5 | 32 | Recuperar automáticamente el buffer tras el timeout |
| 6 | 64 | Hay datos disponibles para recuperar |
| 7 | 128 | **Ocurrió un timeout en el último frame** |

---

## 7. Errores de ejecución

NextDAAD es el único que define y documenta un conjunto completo (`errors.asm:11-13`):

| Código | Significado |
|---|---|
| 0 | Objeto inválido |
| 1 | Localidad inválida |
| 2 | Objeto enviado a la localidad 255 |
| 3 | Profundidad de `PROCESS` excedida |
| 4 | `DOALL` anidado |
| 5 | Opcode ilegal (por ejemplo, un condacto v3 en una base v2) |
| 6 | Proceso inválido |
| 7 | Mensaje o localidad inválidos |
| 8 | `PICTURE` inválido |

PCDAAD emite "Game Error 0-8" con una numeración parecida; ZXDAAD128 añade un "Game error 9"
propio para heap insuficiente.

Diferencia relevante para un compilador: **NextDAAD aborta con error 5 al encontrar un condacto
v3 en una base v2**, mientras que PCDAAD y jDAAD lo ejecutan como si nada y msx2daad consume los
argumentos y retorna. El binario tiene que declarar su versión correctamente.
