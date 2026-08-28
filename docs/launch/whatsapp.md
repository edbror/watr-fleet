# WhatsApp — mensaje reenviable

**Público:** devs y gente que corre agentes de IA para programar (Claude Code, Codex, Gemini, Aider, etc.). Red de Ed.
**Destino:** github.com/edbror/watr-fleet
**Formato:** una sola pieza, reenviable, sin markdown pesado (WhatsApp no lo renderiza). Emojis mínimos.

---

## Versión corta (la reenviable)

Si corres varios agentes de IA a la vez para programar (Claude Code, Codex, Gemini, Aider…), ya conoces el problema: brincas de pane en pane y siempre caes en uno que llevaba 4 minutos trabado esperando que le dijeras y/n, mientras los otros dos quemaban tokens sin que los vieras.

Saqué *Fleet*: un dashboard de terminal (open source, MIT) que te muestra toda tu flota de agentes en una sola pantalla — quién trabaja, quién está bloqueado esperándote, y cuánto llevas gastado. Al bloqueado le contestas y/n sin salir del dashboard.

Trae modo demo, así que lo pruebas en 5 segundos sin nada corriendo:

  brew install edbror/tap/watr-fleet
  fleet --demo

Está en v0.10.4, pre-release — la detección de cada agente todavía es joven. Si corres uno que Fleet lee mal, un issue con cómo se ve tu agente es justo lo que sirve.

github.com/edbror/watr-fleet

---

## Versión ultracorta (para mandar 1-a-1)

¿Corres varios agentes de coding a la vez y nunca sabes cuál está trabado esperándote? Hice Fleet, un dashboard de terminal open source que te lo muestra todo en una pantalla — quién trabaja, quién te espera, cuánto cuesta. Pruébalo en 5 seg:

  brew install edbror/tap/watr-fleet && fleet --demo

github.com/edbror/watr-fleet
