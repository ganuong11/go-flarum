package flarum

// Group group information
type Group struct {
	BaseResources

	NameSingular string `json:"nameSingular"`
	NamePlural   string `json:"namePlural"`
	Color        string `json:"color"`
	Icon         string `json:"icon"`
	IsHidden     bool   `json:"isHidden"`

	// FlarumExtensions []IExtensions
}

// DoInit Initialize Group
func (g *Group) DoInit(id uint64) {
	g.setID(id)
	g.setType("groups")
}

// GetType Get type
func (g *Group) GetType() string {
	return g.Type
}

// // GetAttributes Get attributes
// func (g *Group) GetAttributes() map[string]interface{} {
// 	return nil
// }
