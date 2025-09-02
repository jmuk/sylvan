package tools

import "context"

type writeFileRequest struct {
	Filename string `json:"filename" jsonschema:"required"`
	Content  string `json:"content" jsonschema:"required"`
}

type writeFileResponse struct {
	Ok  bool   `json:"ok" jsonschema:"required"`
	Err string `json:"error,omitempty"`
}

func (ft *FileTools) writeFile(ctx context.Context, req writeFileRequest) writeFileResponse {
	// TODO: ask the user to go or not.
	logger := getLogger(ctx)
	logger.Info("Creating a new file")
	err := ft.root.WriteFile(req.Filename, []byte(req.Content), 0644)
	resp := writeFileResponse{
		Ok: err == nil,
	}
	if err != nil {
		logger.Error("Failed to write", "error", err)
		resp.Err = err.Error()
	}
	return resp
}
