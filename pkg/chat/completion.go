package chat

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"unicode"
)

type completer interface {
	triggerChar(line []rune, pos int) int
	complete(prefix string) []string
}

type commandCompleter struct {
}

var knownCommands = []string{
	"quit",
	"session",
	"mcp",
	"models",
	"commands",
	"help",
	"list-commands",
}

func (cc *commandCompleter) triggerChar(line []rune, pos int) int {
	i := 0
	if len(line) == 0 {
		return -1
	}
	for ; i < pos; i++ {
		if line[i] == '/' {
			break
		} else if !unicode.IsSpace(line[i]) {
			return -1
		} else if i >= pos {
			return -1
		}
	}
	result := i
	for i++; i < pos; i++ {
		if !unicode.IsGraphic(line[i]) || unicode.IsSpace(line[i]) {
			return -1
		}
	}
	return result
}

func (cc *commandCompleter) complete(prefix string) []string {
	// the commandCompleter assumes the prefix includes the / char.
	prefix = prefix[1:]
	results := make([]string, 0, len(knownCommands))
	for _, cmd := range knownCommands {
		if strings.HasPrefix(cmd, prefix) {
			results = append(results, cmd[len(prefix):])
		}
	}
	return results
}

type fileCompleter struct {
	root *os.Root
}

func (fc *fileCompleter) triggerChar(line []rune, pos int) int {
	i := pos - 1
	for ; i >= 0; i-- {
		r := line[i]
		if r == '@' {
			if i == 0 || line[i-1] != '\\' {
				return i
			}
		} else if unicode.IsSpace(r) {
			if i == 0 || line[i-1] != '\\' {
				return -1
			}
		} else if !unicode.IsGraphic(r) {
			return -1
		}
	}
	return -1
}

func (fc *fileCompleter) complete(prefix string) []string {
	// the prefix should start with '@'
	dir, file := filepath.Split(prefix[1:])
	if dir == "" {
		dir = "."
	}
	for strings.HasSuffix(dir, "/") {
		dir = dir[:len(dir)-1]
	}
	ents, err := fs.ReadDir(fc.root.FS(), dir)
	if err != nil {
		return nil
	}
	var results []string
	for _, ent := range ents {
		name := ent.Name()
		if strings.HasPrefix(name, file) {
			results = append(results, name[len(file):])
		}
	}
	return results
}

type combinedCompleter struct {
	comps []completer
}

func (c *combinedCompleter) Do(line []rune, pos int) (newLine [][]rune, length int) {
	for _, cc := range c.comps {
		start := cc.triggerChar(line, pos)
		if start < 0 || start > pos {
			continue
		}
		length = pos - start
		prefix := string(line[start:pos])
		for _, result := range cc.complete(prefix) {
			newLine = append(newLine, []rune(result))
		}
		return newLine, length
	}
	return nil, 0
}

func newCombinedCompleter(root *os.Root) *combinedCompleter {
	return &combinedCompleter{
		comps: []completer{
			&commandCompleter{},
			&fileCompleter{root},
		},
	}
}
