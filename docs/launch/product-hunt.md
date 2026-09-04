# Product Hunt — Fleet

**Realidad primero:** PH no es HN. La audiencia es más product/maker y menos terminal; una TUI de Go convierte peor aquí que un SaaS con landing bonita. Vale la pena por el backlink permanente y por el tráfico de la newsletter, no porque vaya a ser #1.

**Mecánica:** el día arranca a las **00:01 PT** y el ranking es por día. Lanzar el mismo día que el Show HN parte tu atención en dos — no puedes estar contestando comentarios en los dos frentes a la vez. Recomendación: **HN hoy, PH el martes siguiente**, con el momentum y los issues de HN ya digeridos.

---

## Name

```
Fleet
```

## Tagline  (60 caracteres máx.)

```
Every AI coding agent you run, on one terminal dashboard
```
*55 caracteres.*

Alternativas si esa no pasa:
```
htop for your fleet of AI coding agents
```
```
See which AI agent is blocked, working, or burning tokens
```

## Description  (260 caracteres máx.)

```
Running Claude Code, Codex and Aider at once? Fleet is a terminal dashboard that shows who's working, who's blocked waiting on you, and what it all costs. Answer y/n without switching panes. Go + Bubble Tea, MIT, single binary. Try it with `fleet --demo`.
```
*252 caracteres.*

## Links

- **Website:** `https://github.com/edbror/watr-fleet` (el repo ES la landing)
- **GitHub:** mismo

## Topics

`Developer Tools` · `Artificial Intelligence` · `Open Source` · `Productivity` · `Terminal`

*Los primeros tres cargan el peso. "Terminal" es nicho pero atrae justo a quien va a instalarlo.*

## Gallery

Orden importa — PH corta después de la primera imagen en el feed:

1. `docs/opensea.gif` — modo OPEN SEA. Es lo único que se ve distinto a cualquier otra TUI. Va primero.
2. `docs/dashboard.png` — la vista con números: tokens, costo, presión de contexto.
3. `docs/list.png` — la lista de sesiones, lo que ves el 90% del tiempo.

**Ya resuelto:** `docs/list.png` sale a 1270×760 con la banda NEEDS YOU poblada y los tiempos de espera visibles — el demo siembra una sesión bloqueada hace 4 minutos, así que el frame se explica solo sin anotarlo. Regenéralo con `docs/tapes/regen.sh` antes de cada release: el header muestra la versión.

---

## Maker comment  (péganlo tú apenas abra)

```
I built this because I kept losing time to the same thing.

I run several coding agents in parallel — Claude Code refactoring in one tmux pane, Codex writing tests in another, Aider mid-migration in a third. Every few minutes I'd alt-tab into a pane and find it had been sitting on a y/n prompt for four minutes, blocked, doing nothing, while the other two quietly burned tokens.

Session managers show you the panes. They don't tell you which agent needs you, or what the fleet is costing while you look away.

Fleet does three things:

• Attention router — every blocked session in one queue, longest-waiting first. You answer y/n from the dashboard without switching panes.
• Telemetry — real tokens, cost, burn rate, context-window pressure. htop for agents.
• Dashboard mode — token flow and cost distribution across the whole fleet.

Where I have to be straight with you: it identifies 24+ agent CLIs through tmux and by walking the process tree, so a node- or python-launched agent gets identified by the script it's running instead of hiding behind the interpreter. But the deep telemetry — exact tokens, cost, context — comes from Claude Code today, because its transcripts sit on disk and are parseable. For everything else you get presence and activity now. Grok Build's headless JSON, OpenCode's server API and ACP as a universal layer are the roadmap. I'd rather draw that line than blur it.

You can try it with nothing running:

  brew install edbror/tap/watr-fleet
  fleet --demo

Go + Bubble Tea + Lip Gloss, MIT, single static binary.

It's v0.10.5 and pre-release. The detection heuristics are the young part — every agent signals "I'm blocked" differently. If you run one Fleet reads wrong, an issue describing your agent's actual prompt patterns is the most useful thing you can send me.
```

---

## Respuestas rápidas para comentarios

**"¿En qué se diferencia de tmux / zellij / sesh?"**
> Esos manejan sesiones — crearlas, nombrarlas, saltar entre ellas. Fleet no compite ahí; de hecho corre encima de tmux. La diferencia es que responde dos preguntas que un gestor de sesiones no se plantea: cuál de tus agentes está trabado esperándote ahora mismo, y cuánto te está costando la flota mientras no la ves.

**"¿Soporta [agente X]?"**
> Presencia y actividad, casi seguro — detecta 24+ CLIs por nombre de proceso y recorriendo el árbol de procesos. Telemetría profunda (tokens, costo, contexto exacto), hoy solo Claude Code. Si corres X, ábreme un issue con cómo se ve tu prompt de confirmación: eso es exactamente lo que necesito para mapearlo bien.

**"¿Manda mis datos a algún lado?"**
> No. Lee tmux y archivos locales, y no hace ninguna llamada de red. Binario estático, `CGO_ENABLED=0`, MIT — el código está completo en el repo.

**"¿Windows?"**
> Hoy no: depende de tmux. macOS y Linux, arm64 y amd64. WSL debería funcionar, pero no lo he probado — si alguien lo prueba, me interesa el reporte.
