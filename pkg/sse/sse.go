package sse

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"strings"
)

type Event struct {
	Event string `json:"event"`
	Data  string `json:"data"`
	ID    string `json:"id"`
}

type Scanner struct {
	scanner *bufio.Scanner
}

func NewScanner(r io.Reader) *Scanner {
	return &Scanner{scanner: bufio.NewScanner(r)}
}

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
		data := l[(colonPos + 1):]
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
