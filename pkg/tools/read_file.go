package tools

import (
	"context"
	"fmt"
	"io"
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

func (ft *FileTools) readFile(ctx context.Context, req readFileRequest) readFileResponse {
	logger := getLogger(ctx)
	logger.Debug("Reading file")
	fmt.Println("Reading", req.Filename)
	if req.Offset == 0 && req.Length <= 0 {
		data, err := ft.root.ReadFile(req.Filename)
		if err != nil {
			logger.Error("Failed to read", "error", err)
			return readFileResponse{
				Error: err.Error(),
			}
		}
		return readFileResponse{
			Content:     string(data),
			TotalLength: int64(len(data)),
		}
	}

	s, err := ft.root.Stat(req.Filename)
	if err != nil {
		logger.Error("Failed to stat", "error", err)
		return readFileResponse{
			Error: err.Error(),
		}
	}
	totalLength := s.Size()

	f, err := ft.root.Open(req.Filename)
	if err != nil {
		logger.Error("Failed to open", "error", err)
		return readFileResponse{
			Error: err.Error(),
		}
	}
	defer f.Close()
	if req.Offset > 0 {
		if _, err := f.Seek(req.Offset, 0); err != nil {
			logger.Error("Failed to seek", "error", err)
			return readFileResponse{
				Error: err.Error(),
			}
		}
	}
	if req.Length <= 0 {
		data, err := io.ReadAll(f)
		if err != nil {
			logger.Error("Failed to read", "error", err)
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
		logger.Error("Failed to read", "error", err)
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
