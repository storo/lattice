# VHS Demo Script for Lattice
# Generate GIF: vhs docs/demo.tape
# Requires: brew install vhs

Output docs/demo.gif

Set FontSize 14
Set Width 900
Set Height 500
Set Theme "Dracula"
Set Padding 20

# Title
Type "# Lattice - Agent Mesh Framework"
Enter
Sleep 1s

# Build
Type "go build -o lattice ./cmd/lattice"
Enter
Sleep 2s

# Start server
Type "./lattice serve &"
Enter
Sleep 2s

# Interactive mode
Type "./lattice interactive"
Enter
Sleep 1s

# Show agents
Type "/agents"
Enter
Sleep 1.5s

# Show status
Type "/status"
Enter
Sleep 1.5s

# Send a message
Type "Hello! What can you help me with?"
Enter
Sleep 3s

# Exit
Type "/quit"
Enter
Sleep 1s

# Done
Type "# Try it: go get github.com/storo/lattice"
Enter
Sleep 2s
