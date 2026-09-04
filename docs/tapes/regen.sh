#!/bin/sh
# Regenera docs/*.png con la version que se va a publicar.
#
#   docs/tapes/regen.sh v0.10.6
#
# Sin argumento usa el ultimo tag. Requiere vhs (brew install vhs).
set -e
cd "$(dirname "$0")/../.."
VER="${1:-$(git describe --tags --abbrev=0)}"
echo "==> compilando fleet estampado $VER"
go build -ldflags "-s -w -X github.com/edbror/watr-fleet/internal/buildinfo.version=$VER" -o .fleet-still .
trap 'rm -f .fleet-still docs/tapes/stills.gif' EXIT
echo "==> generando capturas"
vhs docs/tapes/stills.tape
echo "==> generando el GIF de OPEN SEA (tarda ~1 min)"
vhs docs/tapes/opensea.tape
echo "==> listo:"
ls -lah docs/list.png docs/dashboard.png docs/opensea.gif
