package main

import (
	"fmt"
	"log"
	"runtime"

	"github.com/bloxown/bo3-client/engine/camera"
	"github.com/bloxown/bo3-client/engine/renderer"
	"github.com/bloxown/bo3-client/engine/shared/datamodel"
	inst "github.com/bloxown/bo3-client/engine/shared/instances"
	"github.com/bloxown/bo3-client/engine/shared/network"
	rl "github.com/gen2brain/raylib-go/raylib"
	"github.com/go-gl/mathgl/mgl32"
)

func use(vars ...interface{}) {
	// intentionally empty
}

const (
	width  = 800
	height = 600
)

func init() {
	// raylib requires OS thread for window and OpenGL
	runtime.LockOSThread()
}

func main() {
	// Init raylib
	rl.InitWindow(width, height, "BO3 Go (Go)")
	defer rl.CloseWindow()

	// Enable VSync (optional)
	rl.SetTargetFPS(60)

	// Create renderer
	rend := renderer.NewRenderer(width, height)

	// Create camera
	cam := camera.NewCamera(mgl32.Vec3{0, 0, 3}, mgl32.Vec3{0, 1, 0}, -90.0, 0.0)

	// Set global ambient light
	rend.AddGlobalLight(mgl32.Vec3{0.3, 0.3, 0.4}, 1.0)

	// Set sun light (direction, color, intensity)
	rend.AddSunLight(mgl32.Vec3{-0.5, -1.0, -0.3}, mgl32.Vec3{1.0, 0.9, 0.8}, 0.8)

	// Timing
	lastTime := float32(rl.GetTime())

	// Connect to server
	datamodel.InitManager() // initializes datamodel.Manager (assumed global)
	dm := datamodel.Manager // type inst.InstanceManager

	// Create some stuff

	test := datamodel.CreateInstance("Part", "baseplate")

	if part, ok := test.(*inst.Part); ok {
		log.Println(part.PrimitiveType) // Now it uses Part's method
	} else {
		log.Println("not a Part")
	}
	test.SetParent(datamodel.Manager.GetRoot().FindFirstChildOfClass("Workspace"))
	datamodel.Manager.GetRoot().PrintHierarchy(2)

	// Create network manager
	nm := network.NewNetworkManager(1024)

	connectionStatus := "Being a server..."

	use(connectionStatus) // cuz go is opinionated

	// Register handlers (example: ping from client -> server PType=0, PSub=0x00)
	nm.RegisterHandler(0x00, 0x00, func(dm inst.InstanceManager, payload []byte, c *network.ClientConn) {
		log.Printf("Received ping payload=%q", string(payload))
		// Example: server-side handler might mutate datamodel (but we will only call handlers on main thread)
		c.SendPacket(0x01, 0x00, []byte("pong"))
		//connectionStatus = "Connected!"
	})
	nm.RegisterHandler(0x00, 0x01, func(dm inst.InstanceManager, payload []byte, c *network.ClientConn) {
		log.Printf("Received login payload=%q", string(payload))
		c.SendPacket(0x01, 0x00, []byte("pong"))
		// Send over stuff
		for _, item := range dm.GetRoot().GetDescendants() {
			c.SendPacket(0x01, 0x05, []byte("")) // 0x05 means: Add Item
			// 0x06 means: Edit Item
			// 0x07 means: Delete Item
			// Anything Item has this format for payload, no spaces in here theyre only used for visibility
			// : [item uuid, gotten by item.localId] 0x1D PROP 0x1E VALUE 0x1F PROP2 0x1E VALUE2
		}
	})

	// start server:
	go func() {
		if err := nm.Serve(dm, "0.0.0.0", 3000); err != nil {
			log.Printf("Serve error: %v", err)
		}
	}()

	// Prepare lua

	for !rl.WindowShouldClose() {
		// Delta time
		currentTime := float32(rl.GetTime())
		dt := currentTime - lastTime
		if dt <= 0 {
			dt = 0.0001
		}
		lastTime = currentTime

		// Keyboard input (WASD)
		forward := rl.IsKeyDown(rl.KeyW)
		backward := rl.IsKeyDown(rl.KeyS)
		left := rl.IsKeyDown(rl.KeyA)
		right := rl.IsKeyDown(rl.KeyD)
		cam.ProcessKeyboard(forward, backward, left, right, dt)

		delta := rl.GetMouseDelta()
		cam.ProcessMouse(delta.X, delta.Y)
		windPos := rl.GetWindowPosition()
		rl.SetMousePosition(int(windPos.X), int(windPos.Y))
		// Start frame
		rend.BeginFrame()

		// Render werkzeug
		parts := dm.GetRoot().GetRenderables()
		//log.Print(parts)
		for _, part := range parts {
			//log.Println(part.PrimitiveType)
			rend.PushPrimitiveBlock(
				part.Position,
				part.Size,
				part.GetRotRender(),
				mgl32.Vec4{1, 0, 0, 1},
				part.PrimitiveType,
			)
		}

		// End frame / draw / present
		rlCam := rl.Camera{
			Position: rl.Vector3{X: cam.Position.X(), Y: cam.Position.Y(), Z: cam.Position.Z()},
			Target: rl.Vector3{
				X: cam.Position.X() + cam.Front.X(),
				Y: cam.Position.Y() + cam.Front.Y(),
				Z: cam.Position.Z() + cam.Front.Z(),
			},
			Up:   rl.Vector3{X: cam.Up.X(), Y: cam.Up.Y(), Z: cam.Up.Z()},
			Fovy: cam.FOV,
			//Type: rl.CameraPerspective, // optional
		}
		rend.PushDUIText(
			mgl32.Vec3{0, 10, -5},
			mgl32.Vec4{1, 0, 0, 1},
			fmt.Sprintf("Prims: %d", rend.GetPrimCount()),
		)
		rend.PushDUIText(
			mgl32.Vec3{0, 30, -5},
			mgl32.Vec4{1, 0, 0, 1},
			fmt.Sprintf("Light sources: %d/16384", rend.GetLCount()),
		)
		rend.PushDUIText(
			mgl32.Vec3{0, 50, -5},
			mgl32.Vec4{1, 0, 0, 1},
			fmt.Sprintf("Instances: %d",
				len(datamodel.Manager.GetRoot().GetDescendants()),
			),
		)
		rend.PushDUIText(
			mgl32.Vec3{0, 70, -5},
			mgl32.Vec4{1, 0, 0, 1},
			fmt.Sprintf("Conn status: %s", connectionStatus),
		)
		rend.EndFrame(rlCam)
		// Drain all pending network events and handle them on the main thread:
		for {
			select {
			case ev, ok := <-nm.Events:
				if !ok {
					// events channel closed (shutdown)
					// optionally set a flag to exit
					goto afterNetwork
				}
				// this calls your registered handler synchronously, which can safely mutate datamodel.Manager
				nm.InvokeHandler(ev, dm)
			default:
				// no more events
				goto afterNetwork
			}
		}
	afterNetwork:
		// Push to client
	}
	// Destroy renderer first
}
