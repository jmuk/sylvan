package tools

import (
	"context"
	"fmt"
)

type writeFileRequest struct {
	Filename string `json:"filename" jsonschema:"required"`
	Content  string `json:"content" jsonschema:"required"`
}

type writeFileResponse struct {
	Ok      bool   `json:"ok" jsonschema:"required"`
	Err     string `json:"error,omitempty"`
	Content string `json:"content" jsonschema:"description=the enw content to be saved when the user edits the content by themself or empty"`
}

func (ft *FileTools) writeFile(ctx context.Context, req writeFileRequest) writeFileResponse {
	logger := getLogger(ctx)
	logger.Info("Creating a new file")
	fmt.Printf("Creating a new file %s with the following content...\n", req.Filename)
	fmt.Println("---\n", req.Content)

	result, err := confirm()
	if err != nil {
		logger.Error("Failed to confirm", "error", err)
		return writeFileResponse{
			Ok:  false,
			Err: err.Error(),
		}
	}
	if result == confirmationEdit {
		logger.Info("User wants to edit")
		req.Content, err = userEdit(logger, req.Filename, req.Content)
		if err != nil {
			return writeFileResponse{
				Ok:  false,
				Err: err.Error(),
			}
		}
	} else if result != confirmationYes {
		logger.Info("User rejected to add the file")
		return writeFileResponse{
			Ok:  false,
			Err: "User rejected to write to the file",
		}
	}

	err = ft.root.WriteFile(req.Filename, []byte(req.Content), 0644)
	resp := writeFileResponse{
		Ok: err == nil,
	}
	if err != nil {
		logger.Error("Failed to write", "error", err)
		resp.Err = err.Error()
	}
	if resp.Ok && result == confirmationEdit {
		resp.Content = req.Content
	}
	return resp
}
