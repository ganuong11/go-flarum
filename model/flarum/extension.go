package flarum

// IExtensionsV1 flarum extension
type IExtensionsV1 interface {
	Register()
	SetAttributes(map[string]interface{})
}
