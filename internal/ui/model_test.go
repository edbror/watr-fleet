package ui

import (
	"errors"
	"strings"
	"testing"
)

// Mientras una pasada está en vuelo, el tick no pide otra. Sin esto los polls
// se enciman y la fuente se escribe desde varias goroutines a la vez, que es
// lo que tiraba fleet con "concurrent map writes".
func TestNoSePidenDosSnapshotsALaVez(t *testing.T) {
	m := Model{}
	if cmd := m.snapshotCmd(); cmd == nil {
		t.Fatal("el primer snapshot no se pidió")
	}
	if cmd := m.snapshotCmd(); cmd != nil {
		t.Error("se pidió un segundo snapshot con uno en vuelo")
	}
	modelo, _ := m.Update(snapshotMsg{})
	m = modelo.(Model)
	if cmd := m.snapshotCmd(); cmd == nil {
		t.Error("tras llegar el resultado, el siguiente tick tenía que pedir otro")
	}
}

// La pantalla de confianza de Claude Code capturada tal cual el 2-nov-2026
// (v2.1.316). No es un prompt de permisos: sus opciones no tienen número, así
// que "1" no hace nada, y su Escape cierra Claude Code y devuelve el pane a la
// shell en vez de rechazar una acción. Medido, no supuesto (WE-553).
const puertaDeConfianza = ` Quick safety check: Is this a project you created or one you trust?
 Claude Code'll be able to read, edit, and execute files here.
 Security guide
 ❯ No, exit
   Yes, I trust this folder
 Enter to confirm · Esc to cancel`

func TestNoSeContestaLaPuertaDeConfianza(t *testing.T) {
	if !enLaPuertaDeConfianza(puertaDeConfianza) {
		t.Error("no reconoció la pantalla de confianza")
	}
}

func TestUnPromptDePermisosSiSeContesta(t *testing.T) {
	// El prompt real de 3 opciones, también capturado el 2-nov-2026.
	permiso := ` Do you want to create nota.txt?
 ❯ 1. Yes
   2. Yes, and switch to accept edits (auto-approve file edits and common file commands) for this session (shift+tab)
   3. No
 Esc to cancel · Tab to amend`
	if enLaPuertaDeConfianza(permiso) {
		t.Error("tomó un prompt de permisos por la pantalla de confianza")
	}
}

// Un rechazo que no se envió no es un fallo de red, y decirlo así manda al
// usuario a buscar un problema que no existe.
func TestElAvisoDistingueLaPuertaDeUnFalloDeRed(t *testing.T) {
	puerta := answerFlash(answerResultMsg{project: "aoe_X", err: ErrPuertaDeConfianza})
	if !strings.Contains(puerta, "trust screen") {
		t.Errorf("aviso = %q, tenía que nombrar la pantalla de confianza", puerta)
	}
	red := answerFlash(answerResultMsg{project: "aoe_X", err: errors.New("exit status 1")})
	if !strings.Contains(red, "couldn't reach") {
		t.Errorf("aviso = %q, se esperaba el de inalcanzable", red)
	}
}
