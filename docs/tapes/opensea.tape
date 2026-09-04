# OPEN SEA — el GIF que abre el README y la galeria de Product Hunt.
# Cada barco es una sesion viva; la energia de la ola sigue el consumo real
# de tokens. Se graba solo la vista de mar: el arranque va oculto.
Output "docs/opensea.gif"

Set FontSize 15
Set Width 1270
Set Height 760
Set Padding 24
Set Theme "Catppuccin Mocha"
Set Framerate 12  # el mar es lento; mas cuadros solo engordan el GIF

Hide
Type "./.fleet-still --demo"
Enter
# Que la flota acumule historia antes de entrar al mar.
Sleep 30s
Type "m"
Sleep 2s
Show

Sleep 9s
