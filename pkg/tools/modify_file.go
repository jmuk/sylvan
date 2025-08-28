package tools

import (
	"fmt"
	"log"
	"os"
	"sort"
)

type modification struct {
	Offset   int64  `json:"offset" jsonschema:"required,description=the offset starting point in bytes to be changed"`
	Previous string `json:"previous" jsonschema:"required,description=the content before the modification"`
	Replace  string `json:"replace" jsonschema:"required,description=the new content to replace the previous content"`
}

type modifyFileRequest struct {
	Filename      string         `json:"filename" jsonschema:"required"`
	Modifications []modification `json:"modifications" jsonschema:"the list of changes to make"`
}

type modifyFileResponse struct {
	Ok    bool   `json:"ok" jsonschema:"required,description=whether all the modification has been made or not"`
	Error string `json:"error" jsonschema:"description=the error message for the first modification to fail or empty if it succeeds"`
}

func modifyFile(req modifyFileRequest) modifyFileResponse {
	log.Printf("Modify file %s", req.Filename)
	data, err := os.ReadFile(req.Filename)
	if err != nil {
		log.Printf("Error reading file %s: %v", req.Filename, err)
		return modifyFileResponse{
			Ok:    false,
			Error: err.Error(),
		}
	}
	strData := string(data)

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
		log.Printf("Modifying %d-th modification", m.index)
		if len(data) < int(m.Offset) || m.Offset < 0 {
			log.Printf("Invalid modification offset %d", m.Offset)
			return modifyFileResponse{
				Ok: false,
				Error: fmt.Sprintf(
					"invalid offset %d at %d-th modification", m.Offset, m.index),
			}
		}
		start := int(m.Offset)
		end := start + len(m.Previous)
		substr := strData[start:end]
		if substr != m.Previous {
			log.Printf("Modification %d does not match with the previous", m.Offset)
			return modifyFileResponse{
				Ok: false,
				Error: fmt.Sprintf(
					"previous %s does not match with the content %s at %d-th modification",
					m.Previous, substr, m.index),
			}
		}
		strData = strData[:start] + m.Replace + strData[end:]
	}

	if err := os.WriteFile(req.Filename, []byte(strData), 0644); err != nil {
		return modifyFileResponse{
			Ok:    false,
			Error: err.Error(),
		}
	}
	return modifyFileResponse{
		Ok: true,
	}
}

var modifyFileRef = &ToolDefinition[modifyFileRequest, modifyFileResponse]{
	name:        "modify_file",
	description: "modify the contents of a file",
	proc:        modifyFile,
}
