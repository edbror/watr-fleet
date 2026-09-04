# Regenera las capturas del README y del kit de lanzamiento.
# No correr a mano: usar docs/tapes/regen.sh, que compila un binario
# estampado con la version real. El header de fleet la muestra, y por eso
# las imagenes envejecen con cada release.
#
# 1270x760 es el tamano de galeria de Product Hunt; sirve igual para el README.
Output "docs/tapes/stills.gif"

Set FontSize 15
Set Width 1270
Set Height 760
Set Padding 24
Set Theme "Catppuccin Mocha"

Hide
Type "./.fleet-still --demo"
Enter
Sleep 3s
Show

# Vista de lista: el 90% del tiempo es esto. El demo siembra una sesion
# bloqueada hace 4 min, asi que la cola NEEDS YOU sale poblada sola.
Screenshot "docs/list.png"

# Vista de dashboard. La espera larga es a proposito: TOKEN FLOW dibuja
# historia por tick, y con 3s de vida sale un muniion de 5 barras.
Sleep 30s
Type "d"
Sleep 2s
Screenshot "docs/dashboard.png"
