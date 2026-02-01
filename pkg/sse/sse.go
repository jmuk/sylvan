// package sse provides SSE (server-sent-event) parsing.
package sse

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"strings"
)

// Event is an event sent from the server.
type Event struct {
	// The name of the event.
	Event string `json:"event"`
	// The payload.
	Data string `json:"data"`
	// The ID of the event.
	ID string `json:"id"`
}

// Scanner offers the functionality to receive events from
// the input.
type Scanner struct {
	scanner *bufio.Scanner
}

// NewScanner creates a new scanner instance.
func NewScanner(r io.Reader) *Scanner {
	return &Scanner{scanner: bufio.NewScanner(r)}
}

// Scan reads a new event from the input. It returns
// nil with io.EOF error if it reaches to the end.
func (s *Scanner) Scan() (*Event, error) {
	ev := &Event{}
	var err error
	var read1 bool
	for s.scanner.Scan() {
		l := s.scanner.Text()
		if l == "" {
			break
		}
		read1 = true
		colonPos := strings.Index(l, ":")
		if colonPos < 0 {
			// Invalid format -- still it should keep reading
			// for this block.
			err = errors.Join(err, fmt.Errorf("colon not found: %s", l))
			continue
		}
		if colonPos == 0 {
			// comment.
			continue
		}
		tag := l[:colonPos]
		data := strings.TrimSpace(l[(colonPos + 1):])
		switch tag {
		case "event":
			ev.Event = data
		case "data":
			if ev.Data != "" {
				ev.Data += "\n" + data
			} else {
				ev.Data = data
			}
		case "id":
			ev.ID = data
		default:
			// ignore others.
		}
	}
	if !read1 {
		return nil, io.EOF
	}
	if err != nil {
		return nil, err
	}
	return ev, nil
}
