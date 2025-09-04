package tools

import (
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
)

func userEdit(logger *slog.Logger, filename, content string) (string, error) {
	editor := os.Getenv("VISUAL")
	if editor == "" {
		editor = os.Getenv("EDITOR")
	}
	if editor == "" {
		// Fall back to nano?
		editor = "nano"
	}
	logger = logger.With("editor", editor)
	logger.Debug("userEdit")

	logger.Debug("Preparing the file")
	ext := filepath.Ext(filename)
	f, err := os.CreateTemp("", "*"+ext)
	if err != nil {
		logger.Error("Failed to create a temp file", "error", err)
		return "", err
	}
	name := f.Name()
	defer os.Remove(name)
	logger = logger.With("tempfile", name)

	if _, err := io.WriteString(f, content); err != nil {
		logger.Error("Failed to write the previous content", "error", err)
		return "", err
	}
	if err := f.Close(); err != nil {
		logger.Error("Failed to close", "error", err)
		return "", err
	}

	logger.Debug("Starting the program")
	cmd := exec.Command(editor, name)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		logger.Error("Failed to run the editor")
		return "", err
	}

	newContent, err := os.ReadFile(name)
	if err != nil {
		logger.Error("Failed to read the file")
		return "", err
	}
	return string(newContent), nil
}
