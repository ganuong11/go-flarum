package flarum

import (
	"encoding/json"
	"reflect"
	"strconv"
)

// EResourceType resource types in flarum
// E means enum
type EResourceType string

const (
	// EBaseUser base user
	EBaseUser EResourceType = "baseuser"

	// ECurrentUser current user
	ECurrentUser EResourceType = "current"

	// EBaseDiscussion post base resource
	EBaseDiscussion EResourceType = "base_discussion"

	// EDiscussion post resource
	EDiscussion EResourceType = "discussion"

	// EForum forum information
	EForum EResourceType = "forums"

	// ETAG tag information
	ETAG EResourceType = "tag"

	// EPost comment information
	EPost EResourceType = "post"

	// EGroup information
	EGroup EResourceType = "group"

	// FoFUpload FoF upload files
	EFoFUploadFiles EResourceType = "files"

	EBlogMeta EResourceType = "blogMeta"
)

// IDataBase flarum data
type IDataBase interface {
	// GetAttributes()
	DoInit(uint64)
	GetType() string
	// GetID() uint64
	GetAttributes() (map[string]interface{}, error)
}

// BaseRelation base resource relations in flarum
type BaseRelation struct {
	BaseResources
}

// Struct2Map convert struct to json
/**
 * From https://stackoverflow.com/a/42849112 Maybe this way is not fast, but it must have the least bugs
 * If it becomes a bottleneck, consider optimization
 */
func Struct2Map(obj interface{}) (newMap map[string]interface{}, err error) {
	data, err := json.Marshal(obj) // Convert to a json string
	if err != nil {
		return
	}
	err = json.Unmarshal(data, &newMap) // Convert to a map
	return
}

// -------------  BaseResources ---------------

// BaseResources base resource structure
/**
 * Please do not directly modify the variables in the struct
 */
type BaseResources struct {
	// Issue-5: flarum needs id as string
	id uint64

	ID string `json:"id"`

	Type string `json:"type"`
}

// DoInit empty function, placeholder use
func (r *BaseResources) DoInit() {}

// setID bind ID
func (r *BaseResources) setID(id uint64) {
	r.id = id
	r.ID = strconv.FormatUint(id, 10)
}

// SetType bind type
func (r *BaseResources) setType(t string) {
	r.Type = t
}

// GetID get ID
func (r *BaseResources) GetID() uint64 {
	return r.id
}

// GetType bind type
func (r *BaseResources) GetType() string {
	return r.Type
}

// GetAttributes get the attribute values of the struct
/**
 * The base class will have this function by default, but in theory this function should not be called, just like base resources should not be used
 */
func (r *BaseResources) GetAttributes() (map[string]interface{}, error) {
	panic("Please write your own get attributes")
}

// -------------  BaseResources ---------------

// IRelation some functions it has
type IRelation interface {
	// field, data
	// BindRelation(string, interface{})
}

// RelationDict dictionary form relations
type RelationDict struct {
	Data BaseRelation `json:"data"`
}

// RelationArray array form relations
type RelationArray struct {
	Data []BaseRelation `json:"data"`
}

// Resource flarum resource
type Resource struct {
	BaseResources
	Attributes    IDataBase `json:"attributes"`
	Relationships IRelation `json:"relationships"`
}

// GetAttributes get the attribute values of the struct, the base class will inherit this function
func (r *Resource) GetAttributes() (map[string]interface{}, error) {
	return Struct2Map(r)
}

// Session flarum session data
type Session struct {
	UserID    uint64 `json:"userId"`
	CsrfToken string `json:"csrfToken"`
}

// APIDoc the result that flarum api will return
type APIDoc struct {
	/**
	 * Although it feels not in use, but need to keep
	 * Links currently clickable links:
	 * 		first: home page
	 * 		next: next page
	 * 		prev: previous page
	 */
	Links map[string]string `json:"links"`

	/**
	 * Data The main data returned by API, there is a pit:
	 *    When discussion post information, this variable is array type, the topic collection to be displayed
	 *    But when requesting post comment information, this variable is dictionary type, a topic corresponding to the current comment
	 *
	 *  ALERT: Must use interface{} type here, and can only use SetData function when assigning
	 */
	Data interface{} `json:"data"`

	Included []Resource `json:"included"`
}

// CoreData data that flarum page needs to return
type CoreData struct {
	Resources   []Resource        `json:"resources"`
	Sessions    Session           `json:"session"`
	Locale      string            `json:"locale"`
	Locales     map[string]string `json:"locales"`
	APIDocument APIDoc            `json:"apiDocument"`
}

// NewResource initialize a resource according to type
func NewResource(resourceType EResourceType, id uint64) Resource {
	var obj Resource
	var data IDataBase
	var defaultRelation IRelation

	switch resourceType {
	// golang no need break
	case EBaseUser:
		data = &BaseUser{}
		defaultRelation = &UserRelations{}
	case ECurrentUser:
		data = &CurrentUser{}
		defaultRelation = &UserRelations{}
	case EDiscussion:
		data = &Discussion{}
		defaultRelation = &DiscussionRelations{}
	case EForum:
		data = &Forum{}
		defaultRelation = &ForumRelations{}
	case ETAG:
		data = &Tag{}
		defaultRelation = &TagRelations{}
	case EPost:
		data = &Post{}
		defaultRelation = &PostRelations{}
	case EFoFUploadFiles:
		data = &FlarumFoFFiles{}
		defaultRelation = &PostRelations{}
	case EBlogMeta:
		data = &FlarumBlogMeta{}
		defaultRelation = &BlogMetaRelations{}
	case EGroup:
		data = &Group{}
	}
	data.DoInit(id)
	obj = Resource{
		Attributes:    data,
		Relationships: defaultRelation,
	}
	obj.setID(id)
	obj.setType(data.GetType())
	return obj
}

// newAPIDoc create a new APIDoc object
func newAPIDoc() APIDoc {
	apiDoc := APIDoc{}
	apiDoc.Links = make(map[string]string)
	apiDoc.Data = make([]Resource, 0)
	apiDoc.Included = make([]Resource, 0)
	return apiDoc
}

// NewCoreData create a new CoreData object
// Usage:
//
//	coreData := flarum.NewCoreData()
//	apiDoc := &coreData.APIDocument // Note, what is obtained is a pointer
func NewCoreData() CoreData {
	coreData := CoreData{}
	coreData.APIDocument = newAPIDoc()
	return coreData
}

// NewAdminCoreData create a new CoreData object
// Usage:
//
//	coreData := flarum.NewAdminCoreData()
//	apiDoc := &coreData.APIDocument // Note, what is obtained is a pointer
func NewAdminCoreData() AdminCoreData {
	adminCoreData := AdminCoreData{}
	adminCoreData.APIDocument = newAPIDoc()
	return adminCoreData
}

// SetData set to dictionary type data
/*
 * Follow this issue:
 * 	https://stackoverflow.com/a/56201087
 * We want to get the correct like,
 *   data: []
 */
func (apiDoc *APIDoc) SetData(data interface{}) {
	if resArr, ok := data.([]Resource); ok {
		//  we can't trust the input data, here we do another check
		// and make sure it's empty but with size 0
		if len(resArr) == 0 {
			apiDoc.Data = make([]Resource, 0)
			return
		}
	}

	apiDoc.Data = data
}

// AppendResources add resources
func (apiDoc *APIDoc) AppendResources(res Resource) {
	apiDoc.Included = append(apiDoc.Included, res)
}

// AppendResources add resources
func (coreData *CoreData) AppendResources(res Resource) {
	coreData.APIDocument.AppendResources(res)
	coreData.Resources = append(coreData.Resources, res)
}

// AddCurrentUser add current user information
func (coreData *CoreData) AddCurrentUser(user Resource) {
	coreData.AppendResources(user)
}

// AddSessionData add user's session information, only for csrf
func (coreData *CoreData) AddSessionData(user Resource, csrf string) {
	coreData.Sessions = Session{
		UserID:    user.GetID(),
		CsrfToken: csrf,
	}
}

// BindRelations bind relations
func (r *Resource) BindRelations(field string, data IRelation) {
	reflect.ValueOf(r.Relationships).Elem().FieldByName(field).Set(reflect.ValueOf(data))
}

// InitBaseResources initialize a base resource
func InitBaseResources(id uint64, t string) BaseRelation {
	br := BaseRelation{}
	br.setID(id)
	br.setType(t)
	return br
}
