package instances

import (
	"errors"
	"strings"
	"sync"
)

// InstanceConstructor returns a newly constructed Instance or nil on failure.
// Constructors should return a concrete non-nil Instance when creation succeeds.
// They must not return a typed-nil (e.g. return (*Folder)(nil)) wrapped in the interface.
type InstanceConstructor func() Instance

// InstanceManager is the interface for a registry + factory that creates Instances
// and provides some helper lookups on the tree.
type InstanceManager interface {
	RegisterClass(className string, ctor InstanceConstructor) error
	Create(className string) Instance
	CreateNamed(className, name string) Instance
	GetRoot() Instance
	SetRoot(root Instance)
	FindByPath(path string) Instance // dot-separated path, e.g. "Root.Folder.Child"
	ListRegistered() []string
}

// instanceManager is a concrete implementation of InstanceManager.
type instanceManager struct {
	mu       sync.RWMutex
	registry map[string]InstanceConstructor
	root     Instance
}

// DefaultInstanceManager is a convenient package-level manager used by BaseInstance.Clone().
var DefaultInstanceManager InstanceManager

// NewInstanceManager constructs an empty manager and registers "Instance" base class.
func NewInstanceManager() InstanceManager {
	m := &instanceManager{
		registry: make(map[string]InstanceConstructor),
	}
	// register default base class constructor
	_ = m.RegisterClass("Instance", func() Instance {
		b := NewBaseInstance("Instance")
		b.name = "Instance"
		return b
	})
	return m
}

func init() {
	DefaultInstanceManager = NewInstanceManager()
	// create default root "Root" (class "Instance")
	DefaultInstanceManager.CreateNamed("Instance", "Root")
}

// RegisterClass registers a constructor under className. Overwrites existing registration.
func (m *instanceManager) RegisterClass(className string, ctor InstanceConstructor) error {
	if className == "" || ctor == nil {
		return errors.New("invalid className or constructor")
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.registry[className] = ctor
	return nil
}

// Create creates a new instance using the registered constructor for className.
// Returns nil if none registered or ctor returned nil.
func (m *instanceManager) Create(className string) Instance {
	m.mu.RLock()
	ctor, ok := m.registry[className]
	m.mu.RUnlock()
	if !ok {
		return nil
	}

	inst := ctor()
	if inst == nil {
		// ctor chose to signal failure
		return nil
	}

	// Make sure a BaseInstance-embedding constructor initialized internal state.
	if b, ok := inst.(*BaseInstance); ok {
		if b.children == nil {
			b.children = make([]Instance, 0)
		}
		if b.className == "" {
			b.className = className
		}
	}
	return inst
}

// CreateNamed creates an instance and sets its name.
func (m *instanceManager) CreateNamed(className, name string) Instance {
	inst := m.Create(className)
	if inst == nil {
		return nil
	}
	inst.SetName(name)
	return inst
}

func (m *instanceManager) GetRoot() Instance {
	m.mu.RLock()
	r := m.root
	m.mu.RUnlock()
	if r == nil {
		// create one if missing
		m.mu.Lock()
		if m.root == nil {
			m.root = m.CreateNamed("Instance", "Root")
		}
		r = m.root
		m.mu.Unlock()
	}
	return r
}

func (m *instanceManager) SetRoot(root Instance) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.root = root
}

// FindByPath resolves a dot-separated path. If first segment doesn't match root name,
// attempts to find a child of root with that name.
func (m *instanceManager) FindByPath(path string) Instance {
	if path == "" {
		return nil
	}
	parts := strings.Split(path, ".")
	root := m.GetRoot()
	if root == nil {
		return nil
	}
	cur := root
	// if first segment doesn't match root, try to find it among root children
	if cur.GetName() != parts[0] {
		cur = root.FindFirstChild(parts[0])
		if cur == nil {
			return nil
		}
	}
	for _, part := range parts[1:] {
		if cur == nil {
			return nil
		}
		cur = cur.FindFirstChild(part)
	}
	return cur
}

// ListRegistered returns a copy of registered class names.
func (m *instanceManager) ListRegistered() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]string, 0, len(m.registry))
	for k := range m.registry {
		out = append(out, k)
	}
	return out
}
