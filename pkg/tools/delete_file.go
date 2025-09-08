package tools

import (
	"context"
	"fmt"
)

type deleteFileRequest struct {
	Filename  string `json:"filename" jsonschema:"description=the file to be deleted"`
	Recursive bool   `json:"recursive" jsonschema:"description=delete all the files included recusively if this is true and the file is a directory"`
}

type deleteFileResponse struct {
	Error string `json:"error" jsonschema:"description=the error message if failed or empty otherwise"`
}

func (ft *FileTools) deleteFile(ctx context.Context, req deleteFileRequest) deleteFileResponse {
	logger := getLogger(ctx)
	logger.Info("Deleting a file")
	stat, err := ft.root.Stat(req.Filename)
	if err != nil {
		logger.Error("Failed to get the file information", "error", err)
		return deleteFileResponse{Error: err.Error()}
	}

	fmt.Println("Deleting the file", req.Filename)
	answer, err := confirmWith(false)
	if err != nil {
		logger.Error("Failed to get the answer", "error", err)
		return deleteFileResponse{Error: err.Error()}
	}
	if answer != confirmationYes {
		logger.Error("User declined to delete the file")
		return deleteFileResponse{Error: "user declined to delete the file"}
	}

	if !stat.IsDir() || !req.Recursive {
		if err := ft.root.Remove(req.Filename); err != nil {
			logger.Error("Failed to delete the file", "error", err)
			fmt.Printf("Failed to delete the file: %v\n", err)
			return deleteFileResponse{Error: err.Error()}
		}
		fmt.Println("Deleted.")
		return deleteFileResponse{}
	}

	if err := ft.root.RemoveAll(req.Filename); err != nil {
		logger.Error("Failed to delete the file recursively", "error", err)
		fmt.Printf("Failed to delete the file: %v\n", err)
		return deleteFileResponse{Error: err.Error()}
	}
	fmt.Println("Deleted.")
	return deleteFileResponse{}
}
