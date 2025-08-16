package instances

import (
	"fmt"
	"log"
	"strings"
	"sync"
)

// Instance is the interface all node types must implement.
// Keep method set minimal but sufficient for hierarchy operations.
type Instance interface {
	// Basic
	GetName() string
	SetName(string)
	GetClassName() string

	// Parent/child management
	GetParent() Instance
	SetParent(Instance)
	GetChildren() []Instance
	FindFirstChild(name string) Instance
	FindFirstChildOfClass(name string) Instance
	// Lifecycle
	GetFullName() string
	Destroy()
	Clone() Instance

	// Debug
	PrintHierarchy(depth int)

	// For recursive methods
	GetDescendants() []Instance
	GetAllOfType(className string) []Instance
	GetRenderables() []*Part
}

// BaseInstance is a default implementation you can embed in subclasses.
// It implements Instance and expects subclasses to embed it in the same package.
type BaseInstance struct {
	mu        sync.Mutex // protects children slice and parent pointer for basic concurrency
	name      string
	parent    Instance
	children  []Instance
	className string
	localId   string
	this      Instance // back-reference to the concrete type
}

// NewBaseInstance returns a fully initialized BaseInstance.
// Prefer creating via an InstanceManager if you want subclasses/registry behaviour.
func NewBaseInstance(className string) *BaseInstance {
	return &BaseInstance{
		name:      className,
		className: className,
		children:  make([]Instance, 0),
	}
}

// --- Basic getters/setters ---
func (b *BaseInstance) GetName() string {
	return b.name
}

func (b *BaseInstance) SetName(name string) {
	b.name = name
}

func (b *BaseInstance) GetClassName() string {
	return b.className
}

func (b *BaseInstance) SetId(id string) {
	b.localId = id
}
func (b *BaseInstance) GetId() string {
	return b.localId
}

// --- Parent & children ---
func (b *BaseInstance) GetParent() Instance {
	return b.parent
}

// SetParent sets the parent of this instance. It removes the instance from the previous parent (if any)
// and adds to the new parent's children. This method assumes parents in this package are BaseInstance or
// embed/behave similarly; subclasses from other packages should ensure compatible behaviour.
func (b *BaseInstance) SetParent(parent Instance) {
	b.mu.Lock()
	defer b.mu.Unlock()

	// remove from old parent
	if b.parent != nil {
		// if parent is a *BaseInstance, directly remove; otherwise, try best-effort via parent's children.
		if pbase, ok := b.parent.(*BaseInstance); ok {
			pbase.removeChild(b)
		} else {
			// best-effort: iterate parent's children and rebuild slice without b if parent exposes GetChildren
			children := b.parent.GetChildren()
			newChildren := make([]Instance, 0, len(children))
			for _, c := range children {
				if c != b {
					newChildren = append(newChildren, c)
				}
			}
			// If parent is something else but happens to be *BaseInstance disguised, try to set its children:
			if pbase2, ok2 := b.parent.(*BaseInstance); ok2 {
				pbase2.children = newChildren
			}
		}
	}

	// set new parent
	b.parent = parent

	// add to new parent's children list
	if parent != nil {
		if pbase, ok := parent.(*BaseInstance); ok {
			pbase.addChild(b)
		} else {
			// if parent is not BaseInstance, we rely on parent implementing correct child handling
			// by calling SetParent on the child (we already did), so we stop here.
		}
	}
}

func (b *BaseInstance) GetChildren() []Instance {
	b.mu.Lock()
	defer b.mu.Unlock()
	cp := make([]Instance, len(b.children))
	copy(cp, b.children)
	return cp
}

func (b *BaseInstance) FindFirstChild(name string) Instance {
	b.mu.Lock()
	defer b.mu.Unlock()
	for _, c := range b.children {
		if c.GetName() == name {
			return c
		}
	}
	return nil
}
func (b *BaseInstance) FindFirstChildOfClass(name string) Instance {
	b.mu.Lock()
	defer b.mu.Unlock()
	for _, c := range b.children {
		if c.GetClassName() == name {
			log.Print(c)
			return c
		}
	}
	return nil
}

// --- full name ---
func (b *BaseInstance) GetFullName() string {
	if b.parent == nil {
		return b.name
	}
	return b.parent.GetFullName() + "." + b.name
}

// --- lifecycle ---
func (b *BaseInstance) Destroy() {
	// copy children to avoid mutation while iterating
	b.mu.Lock()
	childrenCopy := make([]Instance, len(b.children))
	copy(childrenCopy, b.children)
	b.mu.Unlock()

	for _, child := range childrenCopy {
		child.Destroy()
	}

	// remove from parent
	if b.parent != nil {
		if pbase, ok := b.parent.(*BaseInstance); ok {
			pbase.removeChild(b)
		}
	}

	// clear
	b.mu.Lock()
	b.parent = nil
	b.children = nil
	b.mu.Unlock()
}

// Clone creates a deep copy of this instance, attempting to preserve concrete subclass type
// by using the package DefaultInstanceManager to construct a new instance of the same class.
// If no manager or constructor exists, falls back to cloning as a BaseInstance.
func (b *BaseInstance) Clone() Instance {
	// prefer using DefaultInstanceManager (may be nil)
	if DefaultInstanceManager != nil {
		inst := DefaultInstanceManager.Create(b.className)
		if inst == nil {
			// fallback to base
			inst = NewBaseInstance(b.className)
		}
		inst.SetName(b.name)

		// clone children
		for _, child := range b.GetChildren() {
			cc := child.Clone()
			if cc != nil {
				cc.SetParent(inst)
			}
		}
		return inst
	}

	// fallback if no manager: clone as base
	clone := NewBaseInstance(b.className)
	clone.name = b.name
	for _, child := range b.GetChildren() {
		cc := child.Clone()
		if cc != nil {
			cc.SetParent(clone)
		}
	}
	return clone
}

// --- helpers (internal) ---
func (b *BaseInstance) addChild(child Instance) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.children = append(b.children, child)
}

func (b *BaseInstance) removeChild(child Instance) {
	b.mu.Lock()
	defer b.mu.Unlock()
	for idx, c := range b.children {
		if c == child {
			b.children = append(b.children[:idx], b.children[idx+1:]...)
			break
		}
	}
}

// PrintHierarchy prints the subtree rooted at this instance.
func (b *BaseInstance) PrintHierarchy(depth int) {
	indent := strings.Repeat("  ", depth)
	fmt.Printf("%s%s (%s)\n", indent, b.GetName(), b.GetClassName())
	for _, child := range b.GetChildren() {
		child.PrintHierarchy(depth + 1)
	}
}

// GetAllOfType recursively collects all instances of a given class name.
// Includes the current instance if it matches.
// Useful for renderers to queue specific subclasses, e.g., "Part".
func (b *BaseInstance) GetAllOfType(className string) []Instance {
	result := []Instance{}

	b.mu.Lock()
	childrenCopy := make([]Instance, len(b.children))
	copy(childrenCopy, b.children)
	b.mu.Unlock()

	// check self: use the full Instance, not just *BaseInstance
	if b.className == className {
		result = append(result, b.this) // <-- needs a "this" back-reference
	}

	for _, child := range childrenCopy {
		if bi, ok := child.(*BaseInstance); ok {
			result = append(result, bi.GetAllOfType(className)...)
		} else {
			result = append(result, getAllOfTypeInterface(child, className)...)
		}
	}

	return result
}

// helper for non-BaseInstance types
func getAllOfTypeInterface(inst Instance, className string) []Instance {
	result := []Instance{}
	if inst.GetClassName() == className {
		result = append(result, inst)
	}
	for _, c := range inst.GetChildren() {
		result = append(result, getAllOfTypeInterface(c, className)...)
	}
	return result
}

// GetDescendants recursively returns all descendants of this instance (children, grandchildren, etc.)
func (b *BaseInstance) GetDescendants() []Instance {
	b.mu.Lock()
	childrenCopy := make([]Instance, len(b.children))
	copy(childrenCopy, b.children)
	b.mu.Unlock()

	var descendants []Instance

	for _, child := range childrenCopy {
		descendants = append(descendants, child) // add the child itself

		// recurse if child is BaseInstance
		if bi, ok := child.(*BaseInstance); ok {
			descendants = append(descendants, bi.GetDescendants()...)
		} else {
			// handle other Instance implementations
			descendants = append(descendants, getDescendantsInterface(child)...)
		}
	}

	return descendants
}

// helper for non-BaseInstance types
func getDescendantsInterface(inst Instance) []Instance {
	var descendants []Instance
	for _, child := range inst.GetChildren() {
		descendants = append(descendants, child)
		descendants = append(descendants, getDescendantsInterface(child)...)
	}
	return descendants
}

// GetRenderables returns a flat slice of all *Part instances in this subtree.
// It copies the children slice under the BaseInstance lock to iterate without holding
// the lock (same pattern as GetAllOfType).
func (b *BaseInstance) GetRenderables() []*Part {
	result := make([]*Part, 0)

	// copy children (thread-safe)
	b.mu.Lock()
	childrenCopy := make([]Instance, len(b.children))
	copy(childrenCopy, b.children)
	b.mu.Unlock()

	// if this node claims to be a Part, try to append the concrete *Part
	if b.className == "Part" {
		if p, ok := b.this.(*Part); ok && p != nil {
			result = append(result, p)
		}
	}

	// recurse children
	for _, child := range childrenCopy {
		if bi, ok := child.(*BaseInstance); ok {
			// recursive fast-path for BaseInstance children
			result = append(result, bi.GetRenderables()...)
		} else if p, ok := child.(*Part); ok && p != nil {
			// non-BaseInstance that is a Part (defensive)
			result = append(result, p)
		}
		// otherwise ignore non-Part Instances
	}

	return result
}
