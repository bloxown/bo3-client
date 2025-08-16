package instances

// Werkzeug is a simple subclass that acts similar to Roblox's Workspace object; the renderer only renders from here
type Werkzeug struct {
	BaseInstance // embed to inherit behaviour
	Meta         string
}

// NewWerkzeug returns a ready-to-use *Werkzeug wrapped as Instance.
func NewWerkzeug() Instance {
	f := &Werkzeug{}
	// initialize embedded BaseInstance fields explicitly
	f.name = "werkzeug"
	f.className = "Workspace" // roblos compat
	f.children = make([]Instance, 0)
	f.this = f
	return f
}

// Optionally add Werkzeug-specific methods:
func (f *Werkzeug) SetMeta(meta string) {
	f.Meta = meta
}

func (f *Werkzeug) GetMeta() string {
	return f.Meta
}
