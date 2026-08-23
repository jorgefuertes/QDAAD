# Investigación formato DAAD v3

Investigación completa del formato de base de datos de DAAD v3 —leyendo código fuente, por
ingeniería inversa y verificando contra binarios reales— para reunir toda la información
necesaria para construir un compilador capaz de generar bases de datos para cualquier intérprete
de la familia.

Toda la información es pública y los repositorios son software libre, así que toda la
investigación es legal.

## Estado: investigación terminada

La documentación está en **[docs/README.md](docs/README.md)**: 14 documentos en español,
con referencia `fichero:línea` en cada afirmación sobre bytes y verificación empírica de la
especificación binaria.

## Fuentes

Clonadas en el directorio `work`:

- **`DRC`** — El compilador de referencia: <https://github.com/Utodev/DRC>
  No es un compilador monolítico. Son **dos programas**: `drf` (frontend en **Free Pascal**),
  que traduce el `.DSF` a **JSON**, y `drb.php` (backend en **PHP**), que enlaza el DDB binario.
  Todo el layout de bytes vive en `drb.php`.
- **`DRC-Next`** — Fork para Next: <https://github.com/absent42/DRC>
  Su `master` es byte a byte idéntico al upstream. Lo suyo está en la rama `origin/nextdaad`:
  14 líneas que añaden el target `NEXTDAAD`. **Con `master` no se puede generar un DDB que
  NextDAAD acepte.**
- **`jDAAD`** — Intérprete en JavaScript: <https://github.com/Utodev/jDAAD>
- **`msx2daad`** — Intérprete para MSX2, soporte completo de v3: <https://github.com/nataliapc/msx2daad>
- **`NextDAAD`** — Intérprete para ZX Spectrum Next, soporte completo de v3: <https://github.com/absent42/NextDAAD>
- **`PCDAAD`** — Intérprete para MS-DOS: <https://github.com/Utodev/PCDAAD>
- **`ZXDAAD128`** — Intérprete para ZX Spectrum 128: <https://github.com/cronomantic/ZXDAAD128>
  **No es DAAD v3.** Escribe versión 3 en la cabecera para marcar su propio formato bancarizado,
  con cabecera de 58 bytes en vez de 34, y no implementa ningún condacto de v3.
- **`TestUnitDAAD`** — Fuente DAAD para probar intérpretes: <http://github.com/Utodev/TestUnitDAAD>
- **`daad-ready`** — Distribución oficial DAAD Ready, versión B (2026-05-15). La fuente principal
  para la generación de medios finales: 20 scripts de build, 40 herramientas de conversión y los
  manuales oficiales en HTML.

`work/_verify` contiene el material de verificación: el frontend construido desde el fuente, la
matriz de DDBs de prueba y los scripts de validación.

## Objetivos y dónde quedaron cubiertos

| Objetivo | Documento |
|---|---|
| Formato general de base de datos | [02](docs/02-formato-ddb.md), [03](docs/03-secciones.md) |
| Flujo de ejecución | [05](docs/05-flujo-ejecucion.md) |
| Cómo funciona el parser, en inglés y en español | [06](docs/06-parser.md) |
| Peculiaridades de los intérpretes | [08](docs/08-interpretes.md) |
| Inclusión de imágenes | [09](docs/09-graficos.md) |
| Inclusión de música | [10](docs/10-audio.md) |
| Qué pertenece exclusivamente a v3 | [07](docs/07-daad-v3.md) |
| Qué intérpretes están preparados para v3 | [08 §3](docs/08-interpretes.md#3-matriz-de-soporte-de-daad-v3) |
| Cómo se generan ejecutables, discos y taps por plataforma | [11](docs/11-build-plataformas.md) |

Añadidos que no estaban en el objetivo inicial pero que hacen falta para escribir un compilador:
la tabla completa de condactos ([04](docs/04-condactos.md)), la gramática del `.DSF` y el formato
JSON intermedio ([12](docs/12-formato-dsf.md)), y las notas de portabilidad con los bugs
detectados ([13](docs/13-portabilidad.md)).

## Decisiones de la investigación

| Tema | Decisión |
|---|---|
| Idioma | Español |
| Estructura | Documentos especializados en `docs/` con índice |
| Enfoque | Especificación de implementación + guía de build + notas de portabilidad |
| Verificación | Empírica: compilar y contrastar cada campo con volcado hexadecimal |
| Herramientas | `fpc` 3.2.2 vía Homebrew; PHP 8.5 del sistema |
| Artefactos | Todo en `work/_verify/`, fuera de los clones de git |

## Hallazgo principal

**DAAD v3 no es un formato nuevo, es un modo de ejecución.** Un mismo fuente compilado con y sin
`-v3` produce 2 ficheros de tamaño idéntico que difieren **en un solo byte, el offset 0**.
No hay paginación ni límites ampliados.

Sí hay **condactos nuevos** —`SETAT`, `INDIR`, `XMES` como condacto estándar y el metacondacto
`GETKEY`—, pero lo que no hay son **opcodes nuevos**: los 3 que estrena v3 (120, 122 y 124)
ocupaban ya huecos `dumb` en la tabla de v2. El espacio de opcodes sigue tope a 128 porque el
bit 7 está ocupado por la indirección, así que v3 no podía crecer, solo rellenar.

El resto de conclusiones, en [docs/README.md](docs/README.md).
