package flarum

type FlarumFoFFiles struct {
	BaseResources

	BaseName    string `json:"baseName"`    // The base name of the file
	UUID        string `json:"uuid"`        // The unique identifier of the file
	BBCode      string `json:"bbcode"`      // The BBCode format of the file
	URL         string `json:"url"`         // The URL of the file
	Shared      bool   `json:"shared"`      // Whether it is shared
	CanViewInfo bool   `json:"canViewInfo"` // Whether the file information can be viewed
	CanHide     bool   `json:"canHide"`     // Whether the file can be hidden
	CanDelete   bool   `json:"canDelete"`   // Whether the file can be deleted
	FileType    string `json:"fileType"`    // File type
}

// DoInit Initialize tags
func (t *FlarumFoFFiles) DoInit(id uint64) {
	t.setID(id)
	t.setType("files")
}

// GetType Get type
func (t *FlarumFoFFiles) GetType() string {
	return t.Type
}

// GetAttributes
func (t *FlarumFoFFiles) GetAttributes() (map[string]interface{}, error) {
	// Return attribute values
	return map[string]interface{}{
		"uuid":        t.UUID,
		"bbcode":      t.BBCode,
		"url":         t.URL,
		"shared":      t.Shared,
		"canViewInfo": t.CanViewInfo,
		"canHide":     t.CanHide,
		"canDelete":   t.CanDelete,
		"type":        t.FileType,
	}, nil
}
