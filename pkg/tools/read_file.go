package tools

import (
	"io"
	"os"
)

type readFileRequest struct {
	Filename string `json:"filename" jsonschema:"required"`
	Offset   int64  `json:"offset" jsonschema:"title=the offset,description=the offset in bytes to start reading"`
	Length   int64  `json:"length" jsonschema:"description=the total length to read in bytes or -1 to read them all"`
}

type readFileResponse struct {
	Content     string `json:"content" jsonschema:"description=the content to be read"`
	Error       string `json:"error" jsonschema:"description=the error message during the read"`
	TotalLength int64  `json:"total_length" jsonschema:"description=the total length of the file in bytes"`
}

func readFile(req readFileRequest) readFileResponse {
	if req.Offset == 0 && req.Length <= 0 {
		data, err := os.ReadFile(req.Filename)
		if err != nil {
			return readFileResponse{
				Error: err.Error(),
			}
		}
		return readFileResponse{
			Content:     string(data),
			TotalLength: int64(len(data)),
		}
	}

	s, err := os.Stat(req.Filename)
	if err != nil {
		return readFileResponse{
			Error: err.Error(),
		}
	}
	totalLength := s.Size()

	f, err := os.Open(req.Filename)
	if err != nil {
		return readFileResponse{
			Error: err.Error(),
		}
	}
	defer f.Close()
	if req.Offset > 0 {
		if _, err := f.Seek(req.Offset, 0); err != nil {
			return readFileResponse{
				Error: err.Error(),
			}
		}
	}
	if req.Length <= 0 {
		data, err := io.ReadAll(f)
		if err != nil {
			return readFileResponse{
				Error: err.Error(),
			}
		}
		return readFileResponse{
			Content:     string(data),
			TotalLength: totalLength,
		}
	}
	buf := make([]byte, req.Length)
	if n, err := f.Read(buf); err != nil && err != io.EOF {
		return readFileResponse{
			Error: err.Error(),
		}
	} else {
		return readFileResponse{
			Content:     string(buf[:n]),
			TotalLength: totalLength,
		}
	}
}

var readFileDef = &ToolDefinition[readFileRequest, readFileResponse]{
	name:        "read_file",
	description: "Read a file",
	proc:        readFile,
}
