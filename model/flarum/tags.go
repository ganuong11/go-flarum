package flarum

// Tag flarum tag information
type Tag struct {
	BaseResources

	Name               string      `json:"name"`
	Description        string      `json:"description"`
	Slug               string      `json:"slug"`
	Color              string      `json:"color"`
	BackgroundURL      string      `json:"backgroundUrl"`
	BackgroundMode     string      `json:"backgroundMode"`
	Icon               string      `json:"icon"`
	DiscussionCount    uint64      `json:"discussionCount"`
	Position           uint64      `json:"position"`
	DefaultSort        interface{} `json:"defaultSort"`
	IsChild            bool        `json:"isChild"`
	IsHidden           bool        `json:"isHidden"`
	LastPostedAt       string      `json:"lastPostedAt"`
	CanStartDiscussion bool        `json:"canStartDiscussion"`
	CanAddToDiscussion bool        `json:"canAddToDiscussion"`
	IsRestricted       bool        `json:"isRestricted"`
}

// TagChildRelations Relationships that the tag has
// Child nodes need to carry parent node information
type TagChildRelations struct {
	LastPostedDiscussion RelationDict `json:"lastPostedDiscussion"`
	Parent               RelationDict `json:"parent"`
}

// TagRelations Relationships that the tag has
type TagRelations struct {
	LastPostedDiscussion RelationDict   `json:"lastPostedDiscussion"`
	Children             []RelationDict `json:"children"`
}

// DoInit Initialize tags
func (t *Tag) DoInit(id uint64) {
	t.setID(id)
	t.setType("tags")
}

// GetType Get type
func (t *Tag) GetType() string {
	return t.Type
}

// // GetAttributes Get attributes
// func (t *Tag) GetAttributes() map[string]interface{} {
// 	return nil
// }
