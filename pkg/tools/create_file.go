package tools

type writeFileRequest struct {
	Filename string `json:"filename" jsonschema:"required"`
	Content  string `json:"content" jsonschema:"required"`
}

type writeFileResponse struct {
	Ok  bool   `json:"ok" jsonschema:"required"`
	Err string `json:"error,omitempty"`
}

func (ft *FileTools) writeFile(req writeFileRequest) writeFileResponse {
	// TODO: ask the user to go or not.
	ft.logger.Info(
		"Creating a new file",
		"filename", req.Filename, "content", req.Content)
	err := ft.root.WriteFile(req.Filename, []byte(req.Content), 0644)
	resp := writeFileResponse{
		Ok: err == nil,
	}
	if err != nil {
		resp.Err = err.Error()
	}
	return resp
}
