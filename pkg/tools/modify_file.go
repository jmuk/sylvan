package tools

import (
	"context"
	"fmt"
	"sort"

	"github.com/andreyvit/diff"
	"github.com/sergi/go-diff/diffmatchpatch"
)

type modification struct {
	Offset  int64  `json:"offset" jsonschema:"required,description=the offset starting point in bytes to be changed"`
	Length  int64  `json:"length" jsonschema:"required,description=the length of the parts to be modified"`
	Replace string `json:"replace" jsonschema:"required,description=the new content to replace the previous content"`
}

type modifyFileRequest struct {
	Filename      string         `json:"filename" jsonschema:"required"`
	Modifications []modification `json:"modifications" jsonschema:"description=the list of changes to make"`
	Diff          string         `json:"diff" jsonschema:"description=the diff format string describing the change to make. Either of diff or modifications need to be specified"`
}

type modifyFileResponse struct {
	Ok         bool   `json:"ok" jsonschema:"required,description=whether all the modification has been made or not"`
	Error      string `json:"error" jsonschema:"description=the error message for the first modification to fail or empty if it succeeds"`
	NewContent string `json:"new_content" jsonschema:"description=the content modified by the user and actually saved"`
}

func (ft *FileTools) modifyFile(ctx context.Context, req modifyFileRequest) modifyFileResponse {
	logger := getLogger(ctx)
	logger.Info("Modify file")
	if len(req.Modifications) == 0 && req.Diff == "" {
		logger.Error("Both modifications and diff are empty")
		return modifyFileResponse{
			Ok:    false,
			Error: "both modifications and diff are empty",
		}
	}
	fmt.Printf("Modifying %s\n", req.Filename)
	data, err := ft.root.ReadFile(req.Filename)
	if err != nil {
		logger.Error("Error reading file", "error", err)
		return modifyFileResponse{
			Ok:    false,
			Error: err.Error(),
		}
	}
	strData := string(data)

	if len(req.Modifications) == 0 {
		type modWithIndex struct {
			modification
			index int
		}
		sortedMods := make([]modWithIndex, 0, len(req.Modifications))
		for i, m := range req.Modifications {
			sortedMods = append(sortedMods, modWithIndex{
				modification: m,
				index:        i,
			})
		}
		sort.Slice(sortedMods, func(i, j int) bool {
			m1 := sortedMods[i]
			m2 := sortedMods[j]
			return m1.Offset > m2.Offset
		})

		// modify from the last for the simplicity of the edit.
		for _, m := range sortedMods {
			mlog := logger.With(
				"index", m.index, "offset", m.Offset,
				"length", m.Length, "replace", m.Replace)
			mlog.Debug("Modification")
			if len(strData) < int(m.Offset) || m.Offset < 0 {
				mlog.Error("Invalid modification")
				return modifyFileResponse{
					Ok: false,
					Error: fmt.Sprintf(
						"invalid offset %d at %d-th modification", m.Offset, m.index),
				}
			}
			start := int(m.Offset)
			end := start + int(m.Length)
			if end > len(strData) {
				mlog.Error("Incalid modification")
				return modifyFileResponse{
					Ok:    false,
					Error: fmt.Sprintf("invalid length %d at %d-th modification", m.Length, m.index),
				}
			}
			strData = strData[:start] + m.Replace + strData[end:]
		}
	} else {
		dmp := diffmatchpatch.New()
		patches, err := dmp.PatchFromText(req.Diff)
		if err != nil {
			logger.Error("Failed to create patches from diff", "error", err)
			return modifyFileResponse{
				Ok:    false,
				Error: err.Error(),
			}
		}
		var applied []bool
		strData, applied = dmp.PatchApply(patches, strData)
		for i, a := range applied {
			if !a {
				logger.Error("Failed to apply hunk", "hunk-id", i)
				return modifyFileResponse{
					Ok:    false,
					Error: fmt.Sprintf("Failed to apply %d-th hunk of diff", i),
				}
			}
		}
	}

	withNewContent := false
	fmt.Println(diff.LineDiff(string(data), strData))
	answer, err := confirm()
	if err != nil {
		logger.Error("Failed to confirm", "error", err)
		return modifyFileResponse{
			Ok:    false,
			Error: err.Error(),
		}
	}
	if answer == confirmationEdit {
		strData, err = userEdit(logger, req.Filename, strData)
		if err != nil {
			logger.Error("Failed to confirm", "error", err)
			return modifyFileResponse{
				Ok:    false,
				Error: err.Error(),
			}
		}
		withNewContent = true
	} else if answer != confirmationYes {
		logger.Error("User declined")
		return modifyFileResponse{
			Ok:    false,
			Error: "User declined to accept the change",
		}
	}

	if err := ft.root.WriteFile(req.Filename, []byte(strData), 0644); err != nil {
		logger.Error("Failed to write the file", "error", err)
		return modifyFileResponse{
			Ok:    false,
			Error: err.Error(),
		}
	}
	resp := modifyFileResponse{
		Ok: true,
	}
	if withNewContent {
		resp.NewContent = strData
	}
	return resp
}
