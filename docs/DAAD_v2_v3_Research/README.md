# Documentación del formato DAAD v3

Especificación del formato de base de datos de DAAD, con el detalle necesario para escribir un
compilador o un intérprete desde cero, y para generar los medios finales de cada plataforma.

Reconstruida por ingeniería inversa a partir del código fuente de un compilador y cinco
intérpretes, y **verificada empíricamente** compilando bases de datos reales y contrastando cada
campo contra el binario.

Todo el software analizado es libre.

---

## Índice

| Documento | Qué responde |
|---|---|
| [01 — Panorama](01-panorama.md) | Qué componentes existen, qué produce cada uno, dónde está cada cosa |
| [02 — Formato del DDB](02-formato-ddb.md) | **La cabecera byte a byte**, punteros, direcciones base, endianness, alineación, límites |
| [03 — Secciones de datos](03-secciones.md) | Tokens, mensajes, vocabulario, objetos, conexiones, procesos |
| [04 — Condactos](04-condactos.md) | Los 128 opcodes, codificación, aridades irregulares, pseudo-condactos |
| [05 — Flujo de ejecución](05-flujo-ejecucion.md) | El bucle del intérprete, pila de procesos, `DOALL`, **tabla de flags de sistema** |
| [06 — El parser](06-parser.md) | Vocabulario, frase lógica, inglés frente a español, enclíticos |
| [07 — Qué es DAAD v3](07-daad-v3.md) | **El delta v2 → v3 completo**, y qué no es v3 |
| [08 — Los intérpretes](08-interpretes.md) | **Matriz de soporte de v3**, peculiaridades y divergencias |
| [09 — Imágenes](09-graficos.md) | Formatos por plataforma, conversores, empaquetado, `GFX`, y el motor vectorial de 8 bits |
| [10 — Sonido y música](10-audio.md) | `BEEP`, `SFX`, `XPLAY`, formatos y motores |
| [11 — Build por plataforma](11-build-plataformas.md) | De `.DSF` a TAP, DSK, D64, ADF, EXE o web |
| [12 — El fuente `.DSF`](12-formato-dsf.md) | Gramática, preprocesador, símbolos, **el JSON intermedio** |
| [13 — Notas de portabilidad](13-portabilidad.md) | **Trampas y bugs** para quien implemente un compilador |
| [14 — Verificación empírica](14-verificacion.md) | Cómo se ha comprobado la especificación y qué encontró |
| [15 — Límites](15-limites.md) | **Rango usable de cada entidad**, reservados, y qué debe validar un compilador |

---

## Por dónde empezar

**Quiero escribir un compilador de DAAD.**
[01](01-panorama.md) → [02](02-formato-ddb.md) → [03](03-secciones.md) →
[04](04-condactos.md) → [07](07-daad-v3.md) → [15](15-limites.md) → [13](13-portabilidad.md).
El 13 es el que evita perder tiempo: recoge las trampas y los bugs conocidos. El 15 da los
rangos que hay que validar, incluidos los tres que DRC no comprueba.

**Quiero escribir un intérprete.**
[02](02-formato-ddb.md) → [03](03-secciones.md) → [05](05-flujo-ejecucion.md) →
[06](06-parser.md) → [04](04-condactos.md) → [08](08-interpretes.md).
El 08 documenta lo que hicieron los cinco intérpretes existentes y dónde discrepan.

**Quiero escribir una aventura.**
[12](12-formato-dsf.md) → [05](05-flujo-ejecucion.md) → [09](09-graficos.md) →
[10](10-audio.md) → [11](11-build-plataformas.md).
La referencia de autoría oficial sigue siendo `work/daad-ready/DOC/doc_es.html`; esta
documentación cubre lo que aquel manual no detalla.

**Solo quiero saber qué es DAAD v3.**
[07](07-daad-v3.md). Está escrito para leerse suelto.

---

## Las cinco conclusiones

Si no vas a leer nada más:

1. **DRC son dos programas, no uno.** `drf` (Free Pascal) traduce el `.DSF` a JSON; `drb.php`
   (PHP) enlaza el DDB binario. **Todo el layout de bytes vive en `drb.php`** y en ningún otro
   sitio. El JSON es una frontera limpia que permite sustituir cualquiera de las dos mitades.

2. **DAAD v3 no es un formato nuevo.** Es un modo de ejecución seleccionado por un byte.
   Un mismo fuente compilado en v2 y en v3 produce 2 ficheros **idénticos salvo el byte 0**.
   Añade condactos —`SETAT`, `INDIR`, `XMES` nativo y el metacondacto `GETKEY`— pero **no
   amplía el espacio de opcodes**: los 3 que estrena rellenan huecos `dumb` que ya existían.
   No podía crecer, porque el bit 7 del opcode está ocupado por la indirección. Tampoco hay
   paginación ni límites ampliados.

3. **El DDB es una imagen plana de memoria de ≤64 KB con punteros absolutos.** No hay
   reubicación. La dirección base depende del target y hay que conocerla al enlazar. El campo
   que todo el mundo llama "longitud del fichero" es en realidad la **dirección final**.

4. **Los cinco intérpretes no son intercambiables.** El soporte de v3 va de completo
   (msx2daad, NextDAAD) a inexistente (ZXDAAD128), y hay divergencias de comportamiento
   confirmadas en el parser, en `SYNONYM`, en `HASAT` y en los flags de detección de plataforma.
   La matriz está en [08](08-interpretes.md).

5. **ZXDAAD128 escribe versión 3 pero no es DAAD v3.** Es un formato bancarizado propio con
   cabecera de 58 bytes. Y aceptaría un DDB v3 legítimo para leerlo como basura.

---

## Convenciones

- Cada afirmación sobre bytes lleva su referencia `fichero:línea`, relativa a `work/`. La
  excepción es `DAAD V3 CAMBIOS.txt`, que vive junto a estos documentos.
- **[V]** marca lo verificado empíricamente sobre binarios generados durante la investigación.
  El detalle está en [14](14-verificacion.md).
- Cuando dos fuentes discrepan se documentan ambas y se dice cuál gana y por qué.
- **Los bugs se listan como bugs**, separados de la especificación, para que nadie los
  reimplemente creyendo que son el comportamiento previsto.

---

## Fuentes

| Repositorio | Commit | Papel |
|---|---|---|
| [DRC](https://github.com/Utodev/DRC) | `e7bb170` | Compilador de referencia |
| [DRC-Next](https://github.com/absent42/DRC) | rama `nextdaad` | Fork con el target NEXTDAAD |
| [jDAAD](https://github.com/Utodev/jDAAD) | `f21ba61` | Intérprete JavaScript |
| [msx2daad](https://github.com/nataliapc/msx2daad) | `afdd6d2` | Intérprete MSX2 |
| [NextDAAD](https://github.com/absent42/NextDAAD) | `2a206be` | Intérprete ZX Spectrum Next |
| [PCDAAD](https://github.com/Utodev/PCDAAD) | `687ef2b` | Intérprete MS-DOS |
| [ZXDAAD128](https://github.com/cronomantic/ZXDAAD128) | `fe714e0` | Intérprete ZX 128K |
| [TestUnitDAAD](https://github.com/Utodev/TestUnitDAAD) | `33f9535` | Suite de pruebas |
| DAAD Ready | kit versión **B** (2026-05-15) | Scripts de build, herramientas y manuales |
| [`DAAD V3 CAMBIOS.txt`](DAAD%20V3%20CAMBIOS.txt) | — | Notas del autor sobre el delta v2 → v3, con el punto de vista del compilador. Es la única fuente **normativa**: donde contradice a un intérprete, manda ella y la discrepancia se documenta |

Los repositorios clonados están en `work/`; el material de verificación, en `work/_verify/`.
