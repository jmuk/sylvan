package tools

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"regexp"
)

type searchFilesRequest struct {
	PathPattern string `json:"path_pattern" jsonschema:"description=the glob pattern to match with the file names -- always start from the current directory"`
	Grep        string `json:"grep" jsonschema:"The regular expression to match with the content of the files"`
}

type fileEntry struct {
	Path  string `json:"path" jsonschema:"required,description=the path from the current directory"`
	IsDir bool   `json:"is_dir" jsonschema:"description=true if it is a directory"`
}

type searchFilesResponse struct {
	Files []fileEntry `json:"files"`
}

func (ft *FileTools) searchFile(ctx context.Context, req searchFilesRequest) (*searchFilesResponse, error) {
	logger := getLogger(ctx)
	logger.Debug("Search file")
	if req.PathPattern == "" && req.Grep == "" {
		return nil, &ToolError{errors.New("either path_pattern or grep needs to be specified")}
	}
	var contentMatch *regexp.Regexp
	if req.Grep != "" {
		fmt.Printf("Searching for %s\n", req.PathPattern)
		var err error
		contentMatch, err = regexp.Compile(req.Grep)
		if err != nil {
			logger.Error("Failed to parse grep", "error", err)
			return nil, &ToolError{err}
		}
	}
	root, err := ft.getRoot()
	if err != nil {
		return nil, err
	}
	if req.PathPattern != "" {
		fmt.Printf("Searching for %s with %s\n", req.PathPattern, req.Grep)
		files, err := fs.Glob(root.FS(), req.PathPattern)
		if err != nil {
			logger.Error("Failed to glob", "error", err)
			return nil, &ToolError{err}
		}
		resp := &searchFilesResponse{
			Files: make([]fileEntry, 0, len(files)),
		}
		for _, file := range files {
			if contentMatch != nil {
				data, err := root.ReadFile(file)
				if err != nil {
					return nil, &ToolError{err}
				}
				if !contentMatch.Match(data) {
					continue
				}
			}
			stat, err := root.Stat(file)
			if err != nil {
				logger.Error("Failed to stat", "filename", file, "error", err)
				return nil, &ToolError{err}
			}
			resp.Files = append(resp.Files, fileEntry{
				Path:  file,
				IsDir: stat.IsDir(),
			})
		}
		logger.Debug("Matched files", "number_of_files", len(resp.Files))
		return resp, nil
	}

	resp := &searchFilesResponse{}
	fmt.Printf("Searching with %s\n", req.Grep)
	err = filepath.WalkDir(".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		data, err := root.ReadFile(path)
		if err != nil {
			logger.Error("Failed to read file", "filename", path, "error", err)
			return err
		}
		if contentMatch.Match(data) {
			resp.Files = append(resp.Files, fileEntry{
				Path:  path,
				IsDir: d.IsDir(),
			})
		}
		return nil
	})
	if err != nil {
		logger.Error("os.Walk failed", "error", err)
		return nil, &ToolError{err}
	}
	return resp, nil
}
