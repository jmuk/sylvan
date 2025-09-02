package tools

import (
	"context"
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
	Error string      `json:"error"`
}

func (ft *FileTools) searchFile(ctx context.Context, req searchFilesRequest) searchFilesResponse {
	logger := getLogger(ctx)
	logger.Debug("Search file")
	if req.PathPattern == "" && req.Grep == "" {
		return searchFilesResponse{
			Error: "either path_pattern or grep needs to be specified",
		}
	}
	var contentMatch *regexp.Regexp
	if req.Grep != "" {
		var err error
		contentMatch, err = regexp.Compile(req.Grep)
		if err != nil {
			logger.Error("Failed to parse grep", "error", err)
			return searchFilesResponse{
				Error: err.Error(),
			}
		}
	}
	if req.PathPattern != "" {
		files, err := fs.Glob(ft.root.FS(), req.PathPattern)
		if err != nil {
			logger.Error("Failed to glob", "error", err)
			return searchFilesResponse{
				Error: err.Error(),
			}
		}
		resp := searchFilesResponse{
			Files: make([]fileEntry, 0, len(files)),
		}
		for _, file := range files {
			if contentMatch != nil {
				data, err := ft.root.ReadFile(file)
				if err != nil {
					return searchFilesResponse{
						Error: err.Error(),
					}
				}
				if !contentMatch.Match(data) {
					continue
				}
			}
			stat, err := ft.root.Stat(file)
			if err != nil {
				logger.Error("Failed to stat", "filename", file, "error", err)
				return searchFilesResponse{
					Error: err.Error(),
				}
			}
			resp.Files = append(resp.Files, fileEntry{
				Path:  file,
				IsDir: stat.IsDir(),
			})
		}
		logger.Debug("Matched files", "number_of_files", len(resp.Files))
		return resp
	}

	resp := searchFilesResponse{}
	err := filepath.WalkDir(".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		data, err := ft.root.ReadFile(path)
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
		return searchFilesResponse{
			Error: err.Error(),
		}
	}
	return resp
}
