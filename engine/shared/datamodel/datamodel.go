package datamodel

import (
	"log"

	inst "github.com/bloxown/bo3-client/engine/shared/instances"
)

// Manager is the global instance manager for this package
var Manager inst.InstanceManager

// InitManager initializes the Manager and registers classes.
// Call this once at the start of your program.
func InitManager() {
	// Step 1: Create a new instance manager
	Manager = inst.NewInstanceManager()

	// Step 2: Register classes
	if err := Manager.RegisterClass("Workspace", func() inst.Instance {
		return inst.NewWerkzeug()
	}); err != nil {
		log.Fatalf("Failed to register class Workspace: %v", err)
	}

	if err := Manager.RegisterClass("DataModel", func() inst.Instance {
		return inst.NewBaseInstance("DataModel")
	}); err != nil {
		log.Fatalf("Failed to register class DataModel: %v", err)
	}

	if err := Manager.RegisterClass("Part", func() inst.Instance {
		return inst.NewPart()
	}); err != nil {
		log.Fatalf("Failed to register class Part: %v", err)
	}

	// Step 3: create a root instance
	root := Manager.CreateNamed("DataModel", "Root")
	Manager.SetRoot(root)

	// Step 4: Instantiate services
	werkzeug := Manager.CreateNamed("Workspace", "Werkzeug")
	werkzeug.SetParent(root)

	log.Println("Datamodel initialized!")
}

// CreateInstance creates a new instance of a registered class
func CreateInstance(className, name string) inst.Instance {
	if Manager == nil {
		log.Fatal("Manager is not initialized. Call InitManager() first.")
	}
	return Manager.CreateNamed(className, name)
}
