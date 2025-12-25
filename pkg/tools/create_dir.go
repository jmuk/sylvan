package tools

import (
	"context"
	"fmt"

	"github.com/manifoldco/promptui"
)

type createDirRequest struct {
	Dirname string `json:"dirname" jsonschema:"description=the name of the directory to be created"`
}

type createDirResponse struct {
}

func (ft *FileTools) createDir(ctx context.Context, req createDirRequest) (*createDirResponse, error) {
	logger := getLogger(ctx)
	logger.Info("Creating a new directory")
	fmt.Printf("Creating a directory %s\n", req.Dirname)

	answer, err := confirmWith(false)
	if err != nil {
		logger.Error("Failed to get the answer", "error", err)
		return nil, err
	}
	if answer != confirmationYes {
		logger.Error("User declined to create the directory")
		msg, err := (&promptui.Prompt{Label: "Tell me why"}).Run()
		if err != nil {
			return nil, err
		}
		return nil, &ToolError{fmt.Errorf("user declined with `%s`", msg)}
	}

	root, err := ft.getRoot()
	if err != nil {
		return nil, err
	}
	if err := root.MkdirAll(req.Dirname, 0755); err != nil {
		logger.Error("Failed to create the directory", "error", err)
		fmt.Println("Failed to create the directory:", err)
		return nil, &ToolError{err}
	}
	return &createDirResponse{}, nil
}
