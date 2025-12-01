package flarum

import "html/template"

// Post flarum comment information
type Post struct {
	BaseResources

	Number      uint64        `json:"number"`
	CreatedAt   string        `json:"createdAt"`
	ContentType string        `json:"contentType"`
	ContentHTML template.HTML `json:"contentHtml"`
	Content     string        `json:"content"`

	IPAddress string `json:"ipAddress"`
	CanEdit   bool   `json:"canEdit"`
	CanDelete bool   `json:"canDelete"`
	CanHide   bool   `json:"canHide"`
	CanFlag   bool   `json:"canFlag"`
	CanLike   bool   `json:"canLike"`

	IsApproved bool `json:"isApproved"`
	CanApprove bool `json:"canApprove"`
}

// PostRelations Relationships that the post has
type PostRelations struct {
	User        RelationDict  `json:"user"`
	Discussion  RelationDict  `json:"discussion"`
	Flags       RelationArray `json:"flags"`
	Likes       RelationArray `json:"likes"`
	MentionedBy RelationArray `json:"mentionedBy"`

	MentionsPosts RelationArray `json:"mentionsPosts"`
}

// DoInit Initialize comment data
func (p *Post) DoInit(id uint64) {
	p.setID(id)
	p.setType("posts")
}

// // GetType Get type
// func (p *Post) GetType() string {
// 	return p.Type
// }

// // GetAttributes Get attributes
// func (p *Post) GetAttributes() map[string]interface{} {
// 	// uObj := obj.(model.User)
// 	// fmt.Println(uObj)
// 	return nil
// }
