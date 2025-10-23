package chat

import (
	"fmt"
	"strings"
)

type command int

const (
	commandNone command = iota
	commandQuit
	commandSession
	commandList
)

func (c *Chat) parseCommand(line string) (command, []string) {
	line = strings.TrimSpace(line)
	if line[0] != '/' {
		return commandNone, nil
	}
	words := strings.Fields(line[1:])
	if len(words) == 0 {
		return commandNone, nil
	}
	command := words[0]
	switch strings.ToLower(command) {
	case "q", "quit":
		return commandQuit, words[1:]
	case "session":
		return commandSession, words[1:]
	case "commands", "help", "list-commands":
		return commandList, words[1:]
	default:
		fmt.Printf("Unknown command %s, ignoring...\n", command)
		return commandNone, nil
	}
}

func (c *Chat) handleListCommand() {
	fmt.Println(`List of possible commands:
- list, commands, help, or ?: this command -- show the list of commands.
- session: choose a new session.
- q, quit: quit this program.
	`)
}
