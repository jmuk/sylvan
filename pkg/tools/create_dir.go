package tools

import (
	"context"
	"fmt"
)

type createDirRequest struct {
	Dirname string `json:"dirname" jsonschema:"description=the name of the directory to be created"`
}

type createDirResponse struct {
	Error string `json:"error" jsonschema:"description=the error message if failed or empty otherwise"`
}

func (ft *FileTools) createDir(ctx context.Context, req createDirRequest) createDirResponse {
	logger := getLogger(ctx)
	logger.Info("Creating a new directory")
	fmt.Printf("Creating a directory %s\n", req.Dirname)

	answer, err := confirmWith(false)
	if err != nil {
		logger.Error("Failed to get the answer", "error", err)
		return createDirResponse{
			Error: err.Error(),
		}
	}
	if answer != confirmationYes {
		logger.Error("User declined to create the directory")
		return createDirResponse{
			Error: "user declined to create the directory",
		}
	}

	if err := ft.root.MkdirAll(req.Dirname, 0755); err != nil {
		logger.Error("Failed to create the directory", "error", err)
		fmt.Println("Failed to create the directory:", err)
		return createDirResponse{
			Error: err.Error(),
		}
	}
	return createDirResponse{}
}
