package flarum

// "github.com/corvofeng/go-flarum/model/flarum"

type FlarumBlogMeta struct {
	BaseResources

	// featuredImage
	// summary
	FeaturedImage string `json:"featuredImage,omitempty"`
	Summary       string `json:"summary,omitempty"`

	// isFeatured
	// isPendingReview
	// isSized
	IsFeatured      bool `json:"isFeatured,omitempty"`      // Whether it is a featured article
	IsPendingReview bool `json:"isPendingReview,omitempty"` // Whether it is pending review
	IsSized         bool `json:"isSized,omitempty"`         // Whether it has been resized
}

type BlogMetaRelations struct {
	LastPostedDiscussion RelationDict   `json:"lastPostedDiscussion"`
	Children             []RelationDict `json:"children"`
}

// DoInit Initialize tags
func (t *FlarumBlogMeta) DoInit(id uint64) {
	t.setID(id)
	t.setType("blogMeta")
}

// GetType Get type
func (t *FlarumBlogMeta) GetType() string {
	return t.Type
}
