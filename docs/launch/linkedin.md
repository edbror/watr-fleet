# LinkedIn — Fleet

**Público:** tu red profesional en México y LatAm — mezcla de devs, fundadores, gente de producto y clientes potenciales de WATR. No todos corren agentes en tmux; la mitad ni sabe qué es tmux.

**Por eso el ángulo cambia.** En HN vendes el detalle técnico. Aquí vendes el problema: *ya estamos delegando trabajo a varias IAs a la vez y nadie sabe cómo supervisarlas.* El producto es la prueba de que lo pensaste, no el titular.

**Formato:** LinkedIn corta a ~3 líneas antes del "ver más". Las primeras dos frases deciden todo. Sin hashtags de relleno; sin "🚀 Emocionado de anunciar".

**Link:** LinkedIn castiga los posts con liga externa en el cuerpo. Pon la liga en el **primer comentario** y en el post di "link en comentarios".

---

## Versión español  (la principal)

```
Llevo meses corriendo tres o cuatro agentes de IA programando al mismo tiempo. Y descubrí que el problema no es que trabajen mal — es que no sé cuál me está esperando.

Entras a una terminal y el agente lleva cuatro minutos detenido en un "¿continúo? y/n". Cuatro minutos sin hacer nada. Mientras, los otros dos siguieron quemando tokens sin que los vieras.

Eso es supervisión de flota, y ninguna herramienta existente lo resuelve. Los gestores de sesiones te muestran las ventanas; no te dicen cuál te necesita ni cuánto llevas gastado.

Así que hice Fleet: un dashboard de terminal que pone toda tu flota en una sola pantalla. Quién trabaja, quién está bloqueado esperándote — en cola, el que lleva más tiempo primero — y cuánto cuesta todo. Al bloqueado le contestas sin cambiar de ventana.

Es open source, MIT, un binario. Lo pruebas en cinco segundos con modo demo, sin instalar nada más.

Lo interesante no es la herramienta. Es lo que implica: si ya estás delegando trabajo real a varias IAs en paralelo, necesitas los mismos instrumentos que cualquier operación con varios trabajadores — saber quién está bloqueado, quién avanza, y cuánto cuesta la hora. Eso apenas empieza a existir.

Está en v0.10.5, pre-release. La parte que menos confío es la detección: cada agente avisa que está bloqueado a su manera. Si corres uno que Fleet lee mal, dímelo — eso es justo lo que necesito.

Link en comentarios.
```

**Primer comentario (tuyo, inmediato):**
```
github.com/edbror/watr-fleet

brew install edbror/tap/watr-fleet && fleet --demo

Construido sobre el stack de Charm (Bubble Tea + Lip Gloss). MIT.
```

---

## Versión inglés  (si publicas en los dos idiomas, sepáralas un día)

```
I've been running three or four AI coding agents in parallel for months. The problem isn't that they work badly — it's that I can't tell which one is waiting on me.

You tab into a terminal and find the agent has been sitting on a "continue? y/n" for four minutes. Four minutes doing nothing. Meanwhile the other two kept burning tokens while you weren't looking.

That's fleet supervision, and nothing solves it today. Session managers show you the windows; they don't tell you which one needs you, or what you've spent.

So I built Fleet: a terminal dashboard that puts the whole fleet on one screen. Who's working, who's blocked waiting on you — queued, longest-waiting first — and what it costs. You answer the blocked one without switching panes.

Open source, MIT, single binary. Demo mode runs in five seconds with nothing else installed.

The tool isn't the interesting part. This is: if you're delegating real work to several AIs at once, you need the same instruments as any operation with multiple workers — who's blocked, who's moving, what the hour costs. That barely exists yet.

v0.10.5, pre-release. The part I trust least is detection: every agent signals "blocked" differently. If you run one Fleet reads wrong, tell me — that's exactly what I need.

Link in comments.
```

---

## Versión corta  (si prefieres algo de 4 líneas)

```
Corro tres agentes de IA programando a la vez. El problema nunca fue que trabajen mal — es que uno lleva cuatro minutos detenido esperando un "y/n" y no me enteré, mientras los otros dos quemaban tokens.

Hice Fleet: un dashboard de terminal que te muestra la flota completa. Quién trabaja, quién te espera, cuánto va costando.

Open source, MIT, se prueba en cinco segundos. Link en comentarios.
```

---

## Qué NO poner

- **"🚀 Emocionado de anunciar"** — es la señal más clara de post ignorable.
- **Métricas que no tienes.** Nada de "usado por X devs" con 40 estrellas. Se verifica en un clic y quema la credibilidad del resto.
- **Hashtags de relleno.** `#AI #innovation #startup` no mueve alcance y sí abarata el post. Si acaso, dos que describan de verdad.
- **Esconder que es pre-release.** Tu red te conoce; la honestidad es lo que hace que el siguiente lanzamiento también lo lean.
