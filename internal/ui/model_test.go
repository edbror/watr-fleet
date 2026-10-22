package ui

import "testing"

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
