package adapter

import (
	"testing"

	"github.com/edbror/watr-fleet/internal/fleet"
)

// fakeSource devuelve un snapshot fijo.
type fakeSource struct {
	nombre string
	ses    []fleet.Session
}

func (f fakeSource) Name() string                       { return f.nombre }
func (f fakeSource) Snapshot() ([]fleet.Session, error) { return f.ses, nil }

// El caso de WE-1159: el pane reporta el cwd del shell y el transcript el del
// agente, que se metió a un subdirectorio. Antes salían dos filas, una con el
// salto y en ceros, otra con la telemetría y sin salto.
func TestMergeDeSubdirectorio(t *testing.T) {
	pane := fakeSource{"tmux", []fleet.Session{{
		ID: "aoe_X:0.0", Agent: fleet.AgentClaude, Status: fleet.StatusWorking,
		Dir: "/repo", Target: "aoe_X:0.0", ContextPct: -1,
	}}}
	transcript := fakeSource{"claude", []fleet.Session{{
		ID: "cc:abc", Agent: fleet.AgentClaude, Status: fleet.StatusIdle,
		Dir: "/repo/docs/salida", Tokens: 1_000, CostUSD: 12.5, ContextPct: 0.4,
	}}}

	ses, err := NewMultiSource(pane, transcript).Snapshot()
	if err != nil {
		t.Fatal(err)
	}
	if len(ses) != 1 {
		t.Fatalf("filas = %d, se esperaba 1: la sesión se partió en dos", len(ses))
	}
	if ses[0].Target == "" {
		t.Error("la fila sobreviviente perdió el salto")
	}
	if ses[0].Tokens != 1_000 || ses[0].CostUSD != 12.5 {
		t.Errorf("telemetría = %d tokens / $%.2f, se esperaba 1000 / $12.50", ses[0].Tokens, ses[0].CostUSD)
	}
}

// Dos panes en el mismo árbol son dos sesiones, no una. Y el transcript del
// que está más abajo se va con el pane más profundo, no con la raíz.
func TestElPrefijoMasProfundoGana(t *testing.T) {
	panes := fakeSource{"tmux", []fleet.Session{
		{ID: "raiz:0.0", Dir: "/repo", Target: "raiz:0.0", ContextPct: -1},
		{ID: "hijo:0.0", Dir: "/repo/api", Target: "hijo:0.0", ContextPct: -1},
	}}
	transcript := fakeSource{"claude", []fleet.Session{
		{ID: "cc:1", Dir: "/repo/api/internal", Tokens: 7, ContextPct: 0.1},
	}}

	ses, err := NewMultiSource(panes, transcript).Snapshot()
	if err != nil {
		t.Fatal(err)
	}
	if len(ses) != 2 {
		t.Fatalf("filas = %d, se esperaban 2", len(ses))
	}
	if ses[0].Tokens != 0 {
		t.Errorf("la raíz se quedó con la telemetría del hijo (%d tokens)", ses[0].Tokens)
	}
	if ses[1].Tokens != 7 {
		t.Errorf("el pane más profundo no recibió la telemetría (%d tokens)", ses[1].Tokens)
	}
}

// Un segundo transcript bajo el mismo pane abre su propia fila en vez de
// sobrescribir los números del primero.
func TestSegundoTranscriptNoPisaAlPrimero(t *testing.T) {
	pane := fakeSource{"tmux", []fleet.Session{
		{ID: "uno:0.0", Dir: "/casa", Target: "uno:0.0", ContextPct: -1},
	}}
	transcripts := fakeSource{"claude", []fleet.Session{
		{ID: "cc:1", Dir: "/casa/a", Tokens: 100, ContextPct: 0.2},
		{ID: "cc:2", Dir: "/casa/b", Tokens: 200, ContextPct: 0.3},
	}}

	ses, err := NewMultiSource(pane, transcripts).Snapshot()
	if err != nil {
		t.Fatal(err)
	}
	if len(ses) != 2 {
		t.Fatalf("filas = %d, se esperaban 2: una sesión se perdió", len(ses))
	}
	if ses[0].Tokens != 100 {
		t.Errorf("el pane quedó con %d tokens, se esperaba 100", ses[0].Tokens)
	}
	if ses[1].Tokens != 200 {
		t.Errorf("la segunda fila quedó con %d tokens, se esperaba 200", ses[1].Tokens)
	}
}

// Un directorio que sólo comparte prefijo de texto no está dentro del otro.
func TestPrefijoDeTextoNoEsSubdirectorio(t *testing.T) {
	if under("/repo-viejo/x", "/repo") {
		t.Error("/repo-viejo/x no está dentro de /repo")
	}
	if !under("/repo/x", "/repo") {
		t.Error("/repo/x sí está dentro de /repo")
	}
	if under("/repo", "/repo") {
		t.Error("un directorio no está dentro de sí mismo")
	}
	if !under("/Users/ed", "/") {
		t.Error("/Users/ed está dentro de /")
	}
}
