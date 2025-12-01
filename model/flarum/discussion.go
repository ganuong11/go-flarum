package flarum

/**
 * Consistent with topic behavior
 *	refer to:
 *		view/flarum/src/Api/Serializer/DiscussionSerializer.php
 *
 * Why is this variable called Discussion in Flarum? This is defined based on the database content:
 *   In the database:
 *      Discussion is an issue
 * 		Post is a comment under the issue
 * 	When users create, they can
 */

// BaseDiscussion Base class
type BaseDiscussion struct {
	BaseResources

	Title string `json:"title"`
	Slug  string `json:"slug"`
}

// Discussion Post or discussion
// view/flarum/migrations/2015_02_24_000000_create_discussions_table.php
type Discussion struct {
	BaseDiscussion

	CommentCount     uint64 `json:"commentCount"`
	ParticipantCount int    `json:"participantCount"`
	LastPostNumber   uint64 `json:"lastPostNumber"`

	// Information of the first comment, usually created by the author
	CreatedAt   string `json:"createdAt"`
	FirstPostID uint64
	FirstUserID uint64

	// Information of the last comment
	// LastPostID   uint64
	LastPostedAt string `json:"lastPostedAt"`
	LastUserID   uint64

	CanDelete bool `json:"canDelete"`
	CanHide   bool `json:"canHide"`
	CanLock   bool `json:"canLock"`
	CanRename bool `json:"canRename"`
	CanReply  bool `json:"canReply"`
	CanSticky bool `json:"canSticky"`
	CanTag    bool `json:"canTag"`

	// IsHidden   bool `json:"isHidden"`
	// IsApproved bool `json:"isApproved"`
	// IsLocked   bool `json:"isLocked"`
	// IsSticky   bool `json:"isSticky"`

	// HiddenAt   string `json:"hiddenAt"`
	// LastReadAt string `json:"lastReadAt"`
	Subscription string `json:"subscription"`

	// #12 TODO: The position where the current user last read
	LastReadPostNumber int `json:"lastReadPostNumber"`
}

// DiscussionRelations Relationships that the post has
type DiscussionRelations struct {
	User           RelationDict `json:"user"` // User who created the post
	FirstPost      RelationDict `json:"firstPost"`
	LastPostedUser RelationDict `json:"lastPostedUser"`
	BlogMeta       RelationDict `json:"blogMeta"`

	Tags  RelationArray `json:"tags"`
	Posts RelationArray `json:"posts"`

	// BlogMetas RelationArray `json:"blogMeta"`
	// LatestViews    RelationArray `json:"latestViews"`
	// RecipientUsers RelationArray `json:"recipientUsers"`

	// OldRecipientUsers  RelationArray `json:"oldRecipientUsers"`
	// RecipientGroups    RelationArray `json:"recipientGroups"`
	// OldRecipientGroups RelationArray `json:"oldRecipientGroups"`
}

// DoInit Initialize a post
func (d *BaseDiscussion) DoInit(id uint64) {
	d.setType("discussions")
	d.setID(id)
	d.Slug = d.ID
}

// GetType Get type
func (d *BaseDiscussion) GetType() string {
	return d.Type
}

// GetID Get ID information
func (d *BaseDiscussion) GetID() uint64 {
	return d.id
}

// // GetAttributes Get attributes
// func (d *BaseDiscussion) GetAttributes() map[string]interface{} {
// 	// uObj := obj.(model.User)
// 	// fmt.Println(uObj)
// 	return nil
// }
