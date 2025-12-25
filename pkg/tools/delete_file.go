package tools

import (
	"context"
	"fmt"

	"github.com/manifoldco/promptui"
)

type deleteFileRequest struct {
	Filename  string `json:"filename" jsonschema:"description=the file to be deleted"`
	Recursive bool   `json:"recursive" jsonschema:"description=delete all the files included recusively if this is true and the file is a directory"`
}

type deleteFileResponse struct {
}

func (ft *FileTools) deleteFile(ctx context.Context, req deleteFileRequest) (*deleteFileResponse, error) {
	logger := getLogger(ctx)
	logger.Info("Deleting a file")
	root, err := ft.getRoot()
	if err != nil {
		return nil, err
	}
	stat, err := root.Stat(req.Filename)
	if err != nil {
		logger.Error("Failed to get the file information", "error", err)
		return nil, &ToolError{err}
	}

	fmt.Println("Deleting the file", req.Filename)
	answer, err := confirmWith(false)
	if err != nil {
		logger.Error("Failed to get the answer", "error", err)
		return nil, err
	}
	if answer != confirmationYes {
		logger.Error("User declined to delete the file")
		msg, err := (&promptui.Prompt{Label: "Tell me why"}).Run()
		if err != nil {
			return nil, err
		}
		return nil, &ToolError{fmt.Errorf("user declined to delete the file: `%s`", msg)}
	}

	if !stat.IsDir() || !req.Recursive {
		if err := root.Remove(req.Filename); err != nil {
			logger.Error("Failed to delete the file", "error", err)
			fmt.Printf("Failed to delete the file: %v\n", err)
			return nil, &ToolError{err}
		}
		fmt.Println("Deleted.")
		return &deleteFileResponse{}, nil
	}

	if err := root.RemoveAll(req.Filename); err != nil {
		logger.Error("Failed to delete the file recursively", "error", err)
		fmt.Printf("Failed to delete the file: %v\n", err)
		return nil, &ToolError{err}
	}
	fmt.Println("Deleted.")
	return &deleteFileResponse{}, nil
}
