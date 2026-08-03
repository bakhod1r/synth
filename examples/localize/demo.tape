# vhs demo.tape  ->  renders demo.gif
# Requires: vhs (github.com/charmbracelet/vhs), run from repo root:
#   vhs examples/localize/demo.tape

Output examples/localize/demo.gif

Set Shell "bash"
Set FontSize 20
Set Width 1100
Set Height 720
Set Padding 24
Set Theme "Catppuccin Mocha"
Set TypingSpeed 45ms

Type "# One struct. One seed. Coherent records per locale."
Enter
Sleep 1s

Type "cat examples/localize/main.go | sed -n '13,33p'"
Enter
Sleep 3s

Type "go run ./examples/localize"
Enter
Sleep 500ms
Sleep 4s

Type "# name, phone, card and the nested address all agree with the region."
Enter
Sleep 3s
