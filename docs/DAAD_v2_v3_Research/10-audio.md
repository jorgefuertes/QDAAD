# 10 — Sonido y música

3 mecanismos distintos con nombres parecidos: `BEEP` (tono simple), `SFX` (todo lo demás, con
semántica **distinta en cada plataforma**) y `XPLAY` (un pseudo-condacto que se expande en
tiempo de compilación).

---

## 1. `BEEP duración, tono`

El condacto más portable. Genera un tono simple. La duración y el tono son bytes.

2 transformaciones que aplica el compilador y que conviene conocer:

**Degradación a `PAUSE`.** Si la frecuencia queda fuera del rango 48–238, el backend
**sustituye el `BEEP` por un `PAUSE`** (`drb.php:898-903`):

```php
if (($condact->Param2<48) || ($condact->Param2>238))
{
    $condact->Opcode = PAUSE_OPCODE;
    $condact->Condact = 'PAUSE';
    $condact->NumParams = 1;
}
```

Es decir, un `BEEP` con tono inválido no da error: se convierte silenciosamente en una pausa.

**Intercambio de parámetros en Spectrum.** El intérprete de ZX espera los parámetros en el orden
contrario, así que el compilador los permuta para ZX, ZX81 y —en la rama del fork— NEXTDAAD
(`drb.php:905-910`). **Un compilador nuevo debe replicar esta permutación o el sonido saldrá
mal solo en Spectrum.**

Además, la duración de `PAUSE` y `BEEP` se escala por target con `getDurationAdjustment`
(`drb.php:1569`). En NextDAAD el factor es 0,6: un segundo son `PAUSE 83`.

**Amiga y Atari ST no soportan `BEEP`** en absoluto, y por tanto tampoco `XPLAY`.

---

## 2. `XPLAY "cadena"` — MML expandido en compilación

`XPLAY` no existe en el binario. El backend interpreta la cadena MML y la **sustituye por una
secuencia de condactos `BEEP`** (`drb.php:912-947`).

Subconjunto de MML admitido:

| Elemento | Significado |
|---|---|
| `A`–`G`, con `#` y número y `.` | Nota, con sostenido, duración y puntillo |
| `L n` | Duración por defecto |
| `R` | Silencio |
| `N n` | Nota por número, 0–96 |
| `O n` | Octava, 1–8 |
| `T n` | Tempo, 32–255 |
| `V n` | Volumen, 0–15 |
| `<` `>` | Bajar / subir una octava |

Valores por defecto: octava 4, volumen 8, duración 4, tempo 120 (`drb.php:915`).

2 detalles de implementación relevantes:

- El tono se ajusta por target con `getPitchAdjustment` (`drb.php:1574`).
- **Si la cadena no produce ningún `BEEP`** —porque el target no soporta sonido— el condacto se
  sustituye por `AT @38`, que es siempre cierto y por tanto inocuo (`drb.php:939-946`). Un
  `XPLAY` en Amiga no da error: desaparece.

Como cada nota se convierte en un condacto, **una melodía larga infla el DDB**. Es una decisión
de diseño que conviene documentar al autor.

---

## 3. `SFX p1, p2` — el condacto polimórfico

`SFX` es donde más divergen las plataformas. El segundo parámetro no significa lo mismo en todas.

### 3.1 PC/DOS, HTML y targets ADP: reproducción de recursos

| p2 | Efecto |
|---|---|
| 1 | Reproducir la muestra `p1` una vez |
| 2 | Reproducirla en bucle |
| 3 | Una vez, con **frecuencia de muestreo en el byte siguiente** del DDB |
| 4 | En bucle, con frecuencia en el byte siguiente |
| 5 | Detener la muestra |
| 6 / 7 | Reproducir música (`.DRO` en PC, `.mp3` en HTML), una vez / en bucle |
| 8 | Detener la música |
| 9 / 10 | Reproducir FLI (PC) o vídeo (HTML) — **desde la versión A esto se hace con `GFX`** |
| 255 | Reproducir la muestra que cargó el último `PICTURE` |

Los símbolos correspondientes los define el frontend automáticamente: `PLAYSFX=1`,
`PLAYSFXL=2`, `PLAYSFXF=3`, `PLAYSFXFL=4`, `STOPSFX=5`, `PLAYDRO=6`, `PLAYDROL=7`, `STOPDRO=8`,
`PLAYFLI=9`, `FPLAYFLIL=10`.

> **Aridad irregular.** Con `p2` igual a 3 o 4, PCDAAD **consume un byte extra** del flujo de
> condactos (`condacts.pas:1590-1600`). En el fuente se escribe con un `#defb` justo después.
> Un lector de DDB que no lo contemple se desincroniza.

### 3.2 MSX1, C64 y CPC: escritura cruda al chip de sonido

```text
SFX <valor>, <registro>
```

Escribe `valor` en el registro indicado del chip. En MSX el rango es 0–13 (PSG AY-3-8910); en
CPC, además, `SFX flagno 14` cambia la paleta, tomando el índice del flag `flagno` (0–15) y el
color de `flagno+1` (0–26).

msx2daad protege el registro 7: fuerza el bit 7 a 1 y el bit 6 a 0
(`daad_platform_msx2.c:1125-1147`).

Es una diferencia semántica de fondo: en estas plataformas `SFX` no reproduce nada, **programa el
hardware**. Un juego portable no puede usar `SFX` sin condicionarlo al target.

### 3.3 NextDAAD

Sistema propio, el más capaz: **Turbo Sound con 3 chips AY (9 canales) más 2 canales PCM**.

| p2 | Efecto |
|---|---|
| 1 / 2 | WAV o efecto AY, una vez / en bucle |
| 3 / 4 | Igual que 1 / 2 |
| 5 | Detener |
| 6 / 7 | Música, una vez / en bucle |
| 8 | Detener música |
| 9 / 10 | Reproducir `nnn.VID` |
| **11–16** | **Extensión propia**: reserva de canal de muestra |

Formatos: `<GAME>.aks` (Arkos Tracker 3, **exactamente 3 PSG o no suena**), `nnn.aks`,
`STREAM_nnn.aks` (streaming desde SD), `<GAME>_FX.aks` (banco de efectos, ≤2048 bytes),
`nnn.WAV` (PCM mono de 8 bits sin signo, 3500–20000 Hz; **15625 Hz recomendado**).

La ranura de una canción son **10208 bytes**; superarla obliga a usar streaming.

`SFX` **no puede interceptarse desde código del autor** en NextDAAD
(`manual/known-differences.md`).

### 3.4 ZXDAAD128

No tiene implementación propia de `SFX`: llama al vector extern en `VECTOR_OFFSET+2`, o
simplemente avanza el puntero si el vector es cero (`ZXDAAD128.bas:4183-4195`). Solo hay beeper,
vía su rutina interna para `BEEP`. Si el autor quiere música, tiene que meter su propio
reproductor en un banco reservado con `-x n` e invocarlo con `EXTERN n 100`.

---

## 4. Motores de sonido por intérprete

| Intérprete | Hardware | Formatos |
|---|---|---|
| **PCDAAD** | SoundBlaster DSP (DMA auto-init de 8 bits, requiere SB 2.0+), Adlib/OPL2/DualOPL2/OPL3, altavoz interno para `BEEP` | `nnn.SFX` (muestras crudas), `nnn.DRO` (DRO v2.0 de Adlib), `nnn.FLI` |
| **jDAAD** | Web Audio API más elementos `<audio>` y `<video>` | `nnn.mp3`, `nnn.mp4` |
| **msx2daad** | PSG AY-3-8910 por los puertos `$A0`/`$A1` | Replayers de Arkos Tracker 2 incluidos (AKG y AKM) |
| **NextDAAD** | 3 × AY (Turbo Sound) + 2 canales PCM | `.aks`, `.WAV` |
| **ZXDAAD128** | Beeper | ninguno propio |
| **Amiga / ST clásicos** | Canales de sonido nativos | `nnn.WAV` empaquetado en `PART1.DAT` |

### El registro de control de PCDAAD: flag 21

Único entre los intérpretes. `PCDAAD/sfx.pas:9-16` y `adlib.pas:18-25`:

| Bit | Significado |
|---|---|
| 0 | Motor de muestras activo |
| 1 | Motor OPL activo |
| 2 | Muestra en bucle |
| 3 | OPL en bucle |
| 4 | **Hay una muestra sonando** |
| 5 | **Hay música OPL sonando** |

Los bits 4 y 5 son de lectura y permiten a la aventura esperar a que termine un sonido.

---

## 5. Formatos de fichero de entrada

| Fichero | Plataforma | Requisitos |
|---|---|---|
| `SOUNDS/nnn.WAV` | Amiga, ST, PC/DOS | **PCM mono de 8 bits**. En PC, máximo 32000 bytes |
| `SOUNDS/nnn.DRO` | PC/DOS | DOSBox Raw OPL, para música (`SFX n 6/7/8`) |
| `SOUNDS/nnn.mp3` | HTML | efectos y música |
| `SOUNDS/EXPERIMENTAL/` | targets ADP | entra vía `DMG create … -audio-8bit` |
| `<GAME>.aks`, `nnn.aks` | ZX Next | Arkos Tracker 3 |

**Frecuencias admitidas** en Amiga, ST y DOS: **5000, 7000, 9500, 15000 y 20000 Hz**. Si el
fichero no coincide con ninguna, se reproduce a la inmediatamente inferior.

Conversores:

- `TOOLS/WAV2SFX/wav2sfx.exe` — `WAV2SFX <entrada.WAV> <salida.SFX>` para PCDAAD. Rechaza
  ficheros que no sean PCM, los estéreo, los que no sean de 8 bits y los mayores de 32000 bytes.
- `TOOLS/img2daad/wav.php` — mapea las frecuencias a las constantes `DMG_5KHZ`, `DMG_7KHZ`,
  `DMG_9_5KHZ`, `DMG_15KHZ`, `DMG_20KHZ`, `DMG_30KHZ` para Amiga y ST.
- `TOOLS/jDAADMultimedia.php` — genera `sounds.js` y `videos.js` a partir de los MP3 y MP4.

---

## 6. Reproducción "clásica" frente a "nueva"

Hay 2 convenciones históricas para reproducir una muestra:

- **Clásica**: `PICTURE n` carga el recurso `n` (que puede ser un sonido, no una imagen) y
  `SFX 0 255` lo reproduce.
- **Nueva** (PC/DOS y HTML): `SFX n 1` directamente.

La convención clásica es la razón de que el manual insista en **no reutilizar el mismo número
para 2 tipos de recurso distintos**: `PICTURE 12` no sabe si busca imagen o sonido.

---

## 7. Resumen para un compilador

1. `BEEP`: permutar parámetros en ZX y ZX81; degradar a `PAUSE` fuera del rango 48–238; escalar
   la duración por target.
2. `XPLAY`: expandir el MML a `BEEP` en tiempo de compilación; si no sale ninguno, emitir
   `AT @38`.
3. `SFX`: **no validar semánticamente el segundo parámetro**; su significado depende del target.
   Sí hay que contemplar el byte extra con `p2` igual a 3 o 4.
4. Los ficheros de sonido son externos al DDB en todas las plataformas.
