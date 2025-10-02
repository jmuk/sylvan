package claude

import (
	"bufio"
	"bytes"
	"encoding/json"
	"os"
)

func (a *Agent) loadHistory() error {
	if a.historyFile == "" {
		return nil
	}

	f, err := os.Open(a.historyFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	defer f.Close()
	s := bufio.NewScanner(f)
	for s.Scan() {
		m := message{}
		if err := json.Unmarshal(s.Bytes(), &m); err != nil {
			return err
		}
		a.history = append(a.history, m)
	}
	return nil
}

func (a *Agent) saveContent(ms []message) error {
	a.logger.Info("save contents", "messages", ms)
	encodes := make([][]byte, 0, len(ms))
	for _, m := range ms {
		encoded, err := json.Marshal(m)
		if err != nil {
			return err
		}
		encodes = append(encodes, encoded)
	}
	finfo, err := os.Stat(a.historyFile)
	if err != nil {
		if os.IsNotExist(err) {
			// Special pattern.
			return os.WriteFile(a.historyFile, bytes.Join(encodes, []byte{'\n'}), 0600)
		}
	}
	f, err := os.OpenFile(a.historyFile, os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	defer f.Close()
	if _, err := f.Seek(finfo.Size(), 0); err != nil {
		return err
	}
	for _, e := range encodes {
		if _, err := f.Write([]byte{'\n'}); err != nil {
			return err
		}
		if _, err := f.Write(e); err != nil {
			return err
		}
	}
	return nil
}
