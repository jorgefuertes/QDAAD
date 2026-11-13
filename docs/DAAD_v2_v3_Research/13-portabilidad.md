# 13 — Notas para escribir un compilador

Lo que necesita saber quien vaya a implementar un compilador de DAAD desde cero: qué varía entre
targets, qué está mal en el compilador actual y qué decisiones hay que tomar conscientemente.

---

## 1. Qué cambia realmente por target

Solo once cosas del binario dependen del destino. Todo lo demás es idéntico.

| # | Qué cambia | Dónde |
|---|---|---|
| 1 | Nibble de máquina del byte `0x01` | `drb.php:1264-1280` |
| 2 | Byte `0x02` (95, o modo de vídeo en MSX2) | `drb.php:1248-1261` |
| 3 | Orden de bytes: big-endian en ST y Amiga | `drb.php:1307-1310` |
| 4 | Alineación a word en PC, ST, Amiga y HTML | `drb.php:1302-1305` |
| 5 | Dirección base | `drb.php:1282-1300` |
| 6 | **Permutación de parámetros de `BEEP` en ZX y ZX81** | `drb.php:905-910` |
| 7 | Escalado de duración de `PAUSE`/`BEEP` y de tono de `XPLAY` | `drb.php:1569`, `1574` |
| 8 | Capacidad de XMessages: 2 KB, 16 KB o 64 KB | `drb.php:420-445` |
| 9 | `XSPLITSCR` solo en CPC y C64 | `drb.php:1011` |
| 10 | Modo depuración solo en ZX y CPC | `drb.php:1797` |
| 11 | Cabeceras de contenedor: `-ch` (Commodore), `-3h` (+3DOS), `.jddb` (HTML) | `drb.php:1396-1509` |

De estos, el número 6 es el más fácil de pasar por alto: **si no permutas los parámetros de
`BEEP` en Spectrum, el sonido sale mal solo en esa plataforma** y no hay ningún aviso.

---

## 2. Las ocho trampas

### 2.1 El word `0x20` no es la longitud

Guarda la dirección **final**. Ver
[02-formato-ddb.md](02-formato-ddb.md#24-word-0x20--no-es-la-longitud-del-fichero). En los
targets de base 0 coincide con la longitud, lo que hace que el error pase desapercibido hasta que
se prueba en Spectrum.

### 2.2 Los punteros son absolutos, no offsets

El DDB es una imagen de memoria. Un compilador tiene que conocer la dirección base *antes* de
emitir, y no hay tabla de reubicación.

### 2.3 El token 0 se descarta y debe medir un byte

PCDAAD salta el primer token con un `+1` fijo. Emitir un token 0 de más de un byte rompe PCDAAD y
jDAAD, y no rompe msx2daad. Ver
[03-secciones.md](03-secciones.md#12-referencia-desde-un-mensaje-y-el-desfase-de-uno).

### 2.4 La aridad no está en el binario

Sin la tabla de condactos no se puede recorrer un bloque. Y hay cuatro casos donde la aridad real
difiere de la declarada: `INPUT`, `EXTERN` con segundo parámetro 3, `SFX` con segundo parámetro
3 o 4, y el marcador de depuración `0xDC`. Ver
[04-condactos.md](04-condactos.md#3-aridades-irregulares).

### 2.5 Los bloques de condactos se solapan

La deduplicación de colas hace que varias entradas apunten a posiciones distintas de la misma
región. No se puede asumir que sean disjuntos.

### 2.6 `INDIR` exige un DDB escribible

La indirección del segundo parámetro se implementa parcheando el propio DDB en RAM. Un
intérprete que pagine el DDB en una ventana de solo lectura necesita el enfoque de NextDAAD:
un valor de sustitución de un solo uso. Ver
[07-daad-v3.md](07-daad-v3.md#3-indirección-del-segundo-parámetro).

### 2.7 El orden físico de las secciones no es el de los punteros

Ver [02-formato-ddb.md](02-formato-ddb.md#3-orden-de-emisión-de-las-secciones). Un lector debe
navegar siempre por la cabecera.

### 2.8 El offset de `XMES` es LSB/MSB aunque el target sea big endian

Los dos parámetros de `XMES` no son un word: son dos parámetros de condacto sueltos, y
`drb.php:849-850` los calcula sin mirar el endianness. En Amiga y ST, donde todo lo demás del DDB
va en big endian, **este offset va al revés que el resto**. Lo mismo en la forma v2 vía `EXTERN`,
con el MSB en el tercer parámetro. Ver
[07-daad-v3.md](07-daad-v3.md#6-xmes-deja-de-depender-de-maluva).

Y el valor tampoco es un desplazamiento dentro de un fichero, sino una dirección lineal sobre la
concatenación de todos los `.XMB`: ver
[11-build-plataformas.md](11-build-plataformas.md#33-xmessages-el-fichero-0xmb).

---

## 3. Bugs confirmados en el compilador

Todos verificados sobre el código de `work/DRC` en el commit `e7bb170`.

| Gravedad | Ubicación | Problema |
|---|---|---|
| **Alta** | `drb.php:290` | `-np` usa `exit` en lugar de `return`. **[V]** Produce un fichero de **60 bytes** (solo la cabecera), en silencio y con código de salida cero |
| Media | `drb.php:1266` vs `1276` | El machine ID `0x0D` para PC/VGA256 es inalcanzable: la comprobación de `PC` devuelve `0x00` antes. **[V]** confirmado sobre el binario |
| Media | `drb.php:1496` | `$subtarget` no está en el ámbito de `prependC64HeaderToDDB`. Inocuo hoy porque C64 y Plus/4 no tienen subtargets |
| Cosmética | `drb.php:67-79`, `1307-1310` | `$littleEndian` e `isLittleEndianPlatform` están semánticamente **invertidos**. Los dos errores se cancelan y la salida es correcta, pero portar el código guiándose por los nombres produce lo contrario |
| Cosmética | `drb.php:1213` | El comentario dice "doble 00"; se escribe un solo byte, que es lo correcto |
| Cosmética | `drb.php:1239` | `echo "Debug: …"` incondicional dentro de `isValidSubtarget` |
| **Alta** | `USintactic.pas:363` | `if (OTXCount > MAX_OBJECTS)` con `MAX_OBJECTS = 256`: **acepta 256 objetos**, pero el contador de la cabecera es un byte y `chr(256)` da 0. Una aventura con 256 objetos compila sin aviso y declara **cero**. Ver [15-limites.md §4](15-limites.md) |
| Media | `USintactic.pas` | **No hay ninguna comprobación del número de localidades ni de procesos.** Los dos contadores son bytes y se desbordan igual, en silencio. Solo objetos, mensajes y etiquetas están validados |
| Media | `USintactic.pas:631` vs `:633` | El tope de **511 caracteres de un xmensaje se comprueba antes** de añadir el `#n` que convierte `XMESSAGE` en `XMES`. Un `XMESSAGE` de 511 caracteres produce hasta 513 bytes almacenados, por encima del `BlockRead` de 511 de PCDAAD y del `fread` de 512 de msx2daad. La compresión lo suele tapar |
| Media | `drb.php:1921-1927` | Con `-x`, un `0.XMB` que pase de 64 KB **trunca los offsets en silencio**: las comprobaciones de `>0xFFFF` cubren solo los xmensajes, y la de `generateXMessages` corre antes de que `-x` añada las tablas de texto |
| Baja | `UMessageList.pas:65` | El límite de 255 cadenas de `XPLAY`/`XDATA` **informa de otra cosa**: el error dice «Too many messages, total messages in MTX, STX and LTX tables…», que no tiene relación. `OtherTX` no está exenta como sí lo está `XTX` |
| **Alta** | `drb.php:684` vs `UJSONExport.pas:180` | **La `ñ` tiene dos códigos distintos según dónde esté**: 27 en el vocabulario y 26 en los mensajes. Igual con `ü`/`Ü` y `ç`/`Ç`. Una palabra del `/VOC` con `ñ` se guarda con un código que no coincide con el que teclea el jugador (`PCDAAD/ibmpc.pas:188` lo convierte al 26), así que **no puede reconocerse** |
| Media | `drb.php:392` | `checkStrings` **no cubre el vocabulario**: solo mensajes, sysmess, localidades y objetos. Bytes mayores de 127 llegan al DDB sin que nadie avise |
| Media | `drb.php:662-665` | La primera rama de `generateVocabulary` está **muerta y además pierde datos**: compara un byte suelto contra cadenas UTF-8 de 2 bytes, así que nunca casa; y si casara, asigna a la variable temporal y no añade nada al resultado |
| Baja | `UJSONExport.pas:229` vs `drb.php:249` vs `ibmpc.pas:216` | **`ß` tiene tres respuestas distintas en la misma cadena de herramientas**: código 127 en `drf`, glifo 163 en la ruta heredada de `drb`, y glifo 163 al teclearlo en PCDAAD. La ruta viva es la de `drf` |
| Baja | `UJSONExport.pas:59-60` | Con `-7`, **`¡` y `¿` se convierten los dos en `#`**, que es el carácter de escape. Si al siguiente le toca ser `A`–`P`, `b`, `e`, `f`, `g`, `k`, `n`, `r`, `s` o `t`, el analizador se lo come como secuencia |
| Baja | `drb.php:353` y `:374` | El colapso `##` → `#` **existe solo en `drb`** y corre **después** de la pasada de `#A`…`#P`. `drf` no lo entiende |
| Cosmética | `drf.pas:255` | `AddSymbol(SymbolList,'HERE',LOC_HERE)` duplicado |
| Cosmética | `drf.pas:386` | `-check-maluva-disabled` imprime "Forced XMessages" |
| Cosmética | `UJSONExport.pas:170-185` | Los comentarios de `ConvertChars` **están desplazados en uno a partir de `ú`**. Los valores emitidos son los correctos; los comentarios, no |

Constantes declaradas y nunca referenciadas, indicio de funcionalidad prevista y no construida:
`MAX_V3_DIRECTION`, `MAX_BLOCKABLE_CONNECTIONS` y `NUM_PREFIX_CONDACTS`
(`UConstants.pas:20-21, 33`).

---

## 4. Bugs confirmados en los intérpretes

Un compilador no puede arreglarlos, pero sí debe decidir qué generar sabiendo que existen.

| Gravedad | Intérprete | Problema | Qué hacer |
|---|---|---|---|
| **Alta** | jDAAD | `V3CODE()` se invoca como propiedad en las 10 llamadas: **se comporta siempre como v3** | Generar siempre `-v3` para HTML |
| **Alta** | ZXDAAD128 | Nunca escribe los flags 29 y 62: **`HASAT GMODE` siempre falla** | No condicionar imágenes a `HASAT GMODE` si se apunta a esa plataforma |
| Media | jDAAD | `limitEnclicitPronouns()` prueba el bit 6 del flag 53 en vez del bit 2. Colisiona con el listado continuo | Evitar depender del bit 2 en HTML |
| Media | PCDAAD | `HASAT`/`HASNAT` aplican el banco alternativo **sin comprobar la versión** | No usar el bit 1 del flag 53 como scratch en bases v2 |
| Media | ZXDAAD128 | Acepta cualquier DDB con versión 3 y byte `0x02` = 95, incluido un v3 legítimo de otro target, y lo lee con una cabecera de 58 bytes sobre datos de 34 | No entregar DDB de DAAD v3 a ZXDAAD128 |
| Media | jDAAD | `_GFX` casos 9 y 10 sin `break`; casos 5 y 6 con referencias a función en vez de llamadas | Evitar `GFX 9/10` en HTML |
| Baja | PCDAAD, jDAAD | `SYNONYM` nunca marca `done`, ni en v2 | No depender del `done` tras `SYNONYM` |
| Baja | msx2daad | Umbral de nombre convertible en 20 en lugar de 40 | Evitar valores de nombre entre 20 y 39 en vocabularios portables |
| Baja | PCDAAD, NextDAAD | `PAUSE 0` / `GETKEY` **no espera a que se suelte la tecla**, como sí pide la especificación. Dos lecturas seguidas pueden devolver la misma pulsación | No encadenar `GETKEY` sin una espera propia |
| Baja | NextDAAD | `fKey2` se pone **siempre a 0** tras `GETKEY`; PCDAAD guarda ahí el byte alto del código | No usar `fKey2` para distinguir teclas extendidas en juegos portables |
| Media | NextDAAD | **Exime a los códigos 16–31 del desplazamiento de `#g`** (`print.asm:130-149`). Un `#gá#t` muestra `á` en Next y el glifo 149 (`ë`) en los otros cuatro | No meter caracteres del rango bajo dentro de un tramo `#g`…`#t` |
| Media | msx2daad | **Sustituye el `@` siempre**, sin comprobar el idioma (`daad_print.c:128`); los otros tres lo condicionan al bit de español | No usar `@` en bases que no sean españolas |
| Baja | PCDAAD, jDAAD | El **`_` dentro de un token se expande a espacio** (`tokens.pas:33`, `jdaad.js:1624`); msx2daad y NextDAAD no lo hacen | No poner `_` en los tokens de compresión |
| Baja | PCDAAD | `PatchStr` comprueba `'\x81'` **dos veces** (`ibmpc.pas:192-193`), así que la `Ü` no se puede teclear | — |

---

## 5. Decisiones de diseño a tomar conscientemente

### 5.1 ¿Reproducir los bugs o corregirlos?

La respuesta no es la misma en todos los casos:

- **Reproducir**: la permutación de `BEEP` en Spectrum, el token 0 de un byte, la ausencia de
  resta de base en el word `0x20`. Son parte del contrato con los intérpretes existentes.
  Cambiarlos rompe binarios.
- **Corregir**: `-np`, la nomenclatura del endianness, el machine ID `0x0D`. Son fallos del
  compilador que no afectan a ningún DDB válido existente.
- **Documentar y avisar**: el umbral 20/40, los flags 29 y 62. El compilador no puede decidir,
  pero sí puede emitir un aviso cuando detecte un uso arriesgado.

### 5.2 ¿Qué target usar como referencia de v3?

**msx2daad y NextDAAD.** Son los únicos con soporte completo, y msx2daad tiene una suite de 323
pruebas sobre emulador. Cuando dos intérpretes discrepan, el criterio que ha ganado
históricamente es el de msx2daad, con NextDAAD arbitrando empíricamente contra el intérprete ZX
original y dejando la transcripción en los comentarios.

### 5.3 ¿Merece la pena mantener la separación frontend/backend?

La arquitectura de DRC —Pascal para el análisis, PHP para el enlazado, JSON entre medias— tiene
una virtud real: **permite sustituir cualquiera de las dos mitades por separado**. El JSON está
documentado en [12-formato-dsf.md](12-formato-dsf.md#9-el-formato-json-intermedio).

Un compilador nuevo puede:

- Reimplementar solo el frontend y seguir usando `drb.php`, que es donde vive todo el
  conocimiento del layout binario y de las peculiaridades por target.
- Reimplementar solo el backend, consumiendo el JSON de `drf`, que ya resuelve el análisis
  semántico, los símbolos y las etiquetas.
- Reimplementar ambos, usando el JSON como formato de contraste durante el desarrollo.

La tercera opción es la que mejor se presta a la verificación: para cualquier fuente, el JSON de
`drf` y el DDB de `drb` sirven como salidas de referencia.

### 5.4 Objetivos que el formato no permite

Conviene descartarlos pronto:

- **Más de 128 condactos.** El bit 7 del opcode está ocupado por la indirección. Cualquier
  extensión pasa por un mecanismo de escape, no por opcodes nuevos.
- **Más de 255 mensajes por tabla, localidades u objetos.** Los contadores son bytes en la
  cabecera. Lo que sí se puede es derramar entre tablas, como hace el modo no-clásico.
- **DDB de más de 64 KB.** El formato es una imagen plana. Superarlo exige un formato bancarizado
  propio, que es exactamente lo que hizo ZXDAAD128 al coste de romper la compatibilidad.
- **Reubicación.** No hay tabla de relocalización; la dirección base se fija al enlazar.

---

## 6. Estrategia de verificación recomendada

La que se ha usado en esta investigación, y que se puede automatizar:

1. Compilar un mismo fuente para una matriz de targets **elegidos por lo que discriminan**:
   uno con base no nula (ZX/128K), uno con alineación (PC/VGA256), uno big-endian (Amiga), uno
   con byte `0x02` especial (MSX2/8_6), uno con cabecera de contenedor (C64 con `-ch`).
2. Validar cada DDB campo a campo contra la especificación: versión, machine ID y bit de idioma,
   contadores, los 12 punteros dentro del fichero tras restar la base, terminadores de sección,
   ofuscación de mensajes y vocabulario, tamaño de entrada de proceso.
3. Compilar un fuente **sin condicionales de versión** en v2 y v3 y comprobar que los binarios
   difieren en un solo byte.
4. Decodificar un mensaje completo aplicando el algoritmo de tokens del intérprete y comprobar
   que reproduce el texto del fuente.

Los 3 primeros pasos detectan errores de layout; el cuarto detecta errores de compresión, que
son los que producen fallos difíciles de diagnosticar.

Los scripts usados están en `work/_verify/`; ver [14-verificacion.md](14-verificacion.md).
