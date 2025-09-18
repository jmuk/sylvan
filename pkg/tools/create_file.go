package tools

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

type writeFileRequest struct {
	Filename string `json:"filename" jsonschema:"required"`
	Content  string `json:"content" jsonschema:"required"`
}

func (ft *FileTools) writeFile(ctx context.Context, req writeFileRequest) (newContent string, err error) {
	logger := getLogger(ctx)
	filename := req.Filename
	content := req.Content
	logger.Info("Creating a new file")
	fmt.Printf("Creating a new file %s with the following content...\n", filename)
	fmt.Println("---\n", content)

	result, err := confirm()
	if err != nil {
		logger.Error("Failed to confirm", "error", err)
		return "", err
	}
	if result == confirmationEdit {
		logger.Info("User wants to edit")
		content, err = userEdit(logger, filename, content)
		if err != nil {
			return "", &ToolError{err}
		}
	} else if result != confirmationYes {
		logger.Info("User rejected to add the file")
		return "", &ToolError{errors.New("user rejected to write to the file")}
	}

	dirname := filepath.Dir(filename)
	if _, err := ft.root.Stat(dirname); os.IsNotExist(err) {
		fmt.Printf("Creating directory %s\n", dirname)
		logger.Info("Creating directory", "dirname", dirname)
		if err := ft.root.MkdirAll(dirname, 0755); err != nil {
			logger.Error("Failed to create directory", "dirname", dirname, "error", err)
			return "", &ToolError{err}
		}
	} else if err != nil {
		logger.Info("Failed to check directory", "dirname", dirname, "error", err)
		return "", &ToolError{err}
	}

	err = ft.root.WriteFile(filename, []byte(content), 0644)
	if err != nil {
		logger.Error("Failed to write", "error", err)
		return "", &ToolError{err}
	}
	if result == confirmationEdit {
		return content, nil
	}
	return "", nil
}
