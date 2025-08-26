package tools

import (
	"fmt"
	"os"
)

type createFileRequest struct {
	Filename string `json:"filename" jsonschema:"required"`
	Content  string `json:"content" jsonschema:"required"`
}

type createFileResponse struct {
	Ok  bool   `json:"ok" jsonschema:"required"`
	Err string `json:"error,omitempty"`
}

func createFile(req createFileRequest) createFileResponse {
	// TODO: ask the user to go or not.
	fmt.Println("Creating a new file", req.Filename)
	fmt.Println("With the following content: ", req.Content)
	err := os.WriteFile(req.Filename, []byte(req.Content), 0644)
	resp := createFileResponse{
		Ok: err == nil,
	}
	if err != nil {
		resp.Err = err.Error()
	}
	return resp
}

var createFileDef = &ToolDefinition[createFileRequest, createFileResponse]{
	name:         "createFile",
	description:  "Create a new file with the given name and the given content",
	requestType:  "createFileRequest",
	responseType: "createFileResponse",
	proc:         createFile,
}
