package chat

import (
	"fmt"
	"sort"
	"time"

	"github.com/jmuk/sylvan/pkg/session"
	"github.com/manifoldco/promptui"
)

func (c *Chat) chooseNewSession() (*session.Session, error) {
	// Choose a new session.
	sessions, err := session.ListSessions(c.cwd)
	if err != nil {
		return nil, err
	}
	if len(sessions) == 0 {
		fmt.Println("No sessions found to select")
		return nil, nil
	}
	sort.Slice(sessions, func(i, j int) bool {
		t1 := sessions[i].Timestamp()
		t2 := sessions[j].Timestamp()
		// Newer one comes earlier.
		return t1.After(t2)
	})
	var foundExisting bool
	for _, s := range sessions {
		if s.ID() == c.cs.s.ID() {
			foundExisting = true
			break
		}
	}
	if !foundExisting {
		sessions = append([]*session.Session{c.cs.s}, sessions...)
	}
	items := make([]string, 0, len(sessions))
	var cursorPos int
	for i, s := range sessions {
		item := fmt.Sprintf("%s at %s", s.ID(), s.Timestamp().Format(time.RFC1123Z))
		if s.ID() == c.cs.s.ID() {
			item += " (current session)"
			cursorPos = i
		}
		items = append(items, item)
	}
	sel := promptui.Select{
		Label:     "Select the session to switch",
		Items:     items,
		CursorPos: cursorPos,
	}
	idx, _, err := sel.Run()
	if err != nil {
		return nil, err
	}
	if sessions[idx].ID() == c.cs.s.ID() {
		return nil, nil
	}
	return sessions[idx], nil
}

func (c *Chat) handleSessionCommands(args []string) (bool, error) {
	var newSession *session.Session
	if len(args) == 0 {
		var err error
		newSession, err = c.chooseNewSession()
		if err != nil {
			return false, err
		}
	} else {
		sessionID := args[0]
		if sessionID == "last" {
			// Choose the last session.
			sessions, err := session.ListSessions(c.cwd)
			if err != nil {
				return false, err
			}
			if sessions[0].ID() != c.cs.s.ID() {
				newSession = sessions[0]
			}
		} else {
			var err error
			newSession, err = session.NewFromID(sessionID)
			if err != nil {
				return false, err
			}
		}
	}
	if newSession == nil {
		return false, nil
	}
	if err := c.cs.Close(); err != nil {
		return false, err
	}
	c.cs = &chatSession{s: newSession}
	fmt.Printf("Session is updated to %s\n", c.cs.s.ID())
	return true, nil
}
