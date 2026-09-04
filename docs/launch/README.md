# Kit de lanzamiento — Fleet

Cinco canales, un producto. Cada archivo trae el texto listo para pegar más las
respuestas a los comentarios previsibles.

| Canal | Archivo | Idioma | Estado |
|---|---|---|---|
| Show HN | `posts-copypaste.md` §1 · estrategia en `hn.md` | EN | listo |
| r/commandline | `posts-copypaste.md` §2 | EN | listo |
| r/golang | `posts-copypaste.md` §3 | EN | listo |
| Product Hunt | `product-hunt.md` | EN | listo |
| LinkedIn | `linkedin.md` | ES + EN | listo |
| WhatsApp | `whatsapp.md` | ES | listo |
| Blog | `blog-post.md` | EN | listo |

## Orden sugerido

No lances todo el mismo día. Cada canal necesita que estés al teclado
contestando; dos a la vez significa contestar mal en los dos.

```
Día 1  ·  Show HN            mañana entre semana, US-Eastern
Día 1  ·  WhatsApp           el mismo día, ya con el post de HN vivo
Día 2  ·  r/commandline      lidera con el GIF
Día 2  ·  LinkedIn           público distinto, no compite con HN
Día 3  ·  r/golang           técnico, sobre el código
Día 7+ ·  Product Hunt       martes, con los issues de HN ya digeridos
```

**Sobre el día:** HN y Product Hunt rinden notablemente menos en viernes y fin
de semana. Martes a jueves por la mañana es la ventana.

## Línea de honestidad — va en todos lados

**v0.10.5, pre-release.** La telemetría profunda (tokens, costo y presión de
contexto reales) hoy sale de Claude Code, porque sus transcripts están en disco
y se pueden parsear. Los otros 24+ CLIs se detectan por tmux: presencia y
actividad. Decirlo de frente gana más de lo que cuesta, y en HN lo contrario se
castiga.

## Verificado el 2026-09-04

- Release publicado, sin draft, con los 4 tarballs (darwin/linux × amd64/arm64) respondiendo **HTTP 200**.
- **Checksum darwin_arm64 cuadra** contra la fórmula del tap.
- El binario publicado corre y trae `-demo`.
- **El comando correcto es `brew install edbror/tap/watr-fleet`** — no `.../fleet`. La fórmula es `watr-fleet.rb`, y Homebrew la nombra por su archivo. Confirmado leyendo el tap.
- `fleet --version` agregado en v0.10.5. Antes imprimía el usage — es lo primero que teclea medio HN después de instalar.

## Abierto

- **La fórmula vive en la raíz** del tap, no en `Formula/`. Homebrew lo acepta, pero `raw.githubusercontent.com/edbror/homebrew-tap/main/Formula/watr-fleet.rb` da 404 — si alguien la busca ahí, no la encuentra. Moverla a `Formula/` es cosmético pero gratis.
- ~~Falta un frame que explique el producto sin leer~~ — hecho. `docs/list.png` a 1270×760 con la cola NEEDS YOU poblada; el demo siembra una sesión bloqueada hace 4 minutos, así que el frame se explica solo sin anotarlo.
- **Las imágenes se regeneran con `docs/tapes/regen.sh <tag>`** (VHS). Hacerlo en cada release: el header de fleet muestra la versión, así que las capturas envejecen solas — `dashboard.png` llevaba mostrando `v0.8`.
- **`tour.gif` y `launcher.png` siguen viejos.** No tienen tape; el tour necesita decidir qué recorrido enseña. Ninguno se usa en el README ni en la galería, así que no bloquean.
- **No hay CI en pull requests.** `release.yml` corre solo con tags: nada valida un PR antes del merge. Un workflow de `go test` es media hora y evita taguear un release roto.
- **Cuenta de Reddit con karma suficiente** para postear en r/commandline y r/golang. Los dos filtran cuentas nuevas.
