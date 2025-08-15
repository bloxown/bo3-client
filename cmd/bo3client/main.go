package main

import (
	"fmt"
	"math"
	"runtime"

	"github.com/bloxown/bo3-client/engine/camera"
	"github.com/bloxown/bo3-client/engine/renderer"
	rl "github.com/gen2brain/raylib-go/raylib"
	"github.com/go-gl/mathgl/mgl32"
)

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

		// Example: rotating cube
		angle := float32((math.Sin(float64(currentTime))*0.5 + 0.5) * math.Pi)
		rot := mgl32.QuatRotate(angle, mgl32.Vec3{0, 1, 0})
		//cam.ProcessMouse(angle/2, angle)
		rend.PushPrimitiveBlock(
			mgl32.Vec3{0, 10, -5},  // position
			mgl32.Vec3{1, 1, 1},    // size
			rot,                    // rotation quaternion
			mgl32.Vec4{1, 0, 0, 1}, // color (red)
			"LightCube",
		)

		// Large floor
		rend.PushPrimitiveBlock(
			mgl32.Vec3{0, -5, -5},
			mgl32.Vec3{100, 1, 100},
			mgl32.QuatIdent(),
			mgl32.Vec4{0, 1, 0, 1},
			"cube",
		)

		// Example: spawn a 3x3x3 grid of cubes
		for x := -1; x <= 1; x++ {
			for y := -1; y <= 1; y++ {
				for z := -1; z <= 1; z++ {
					pos := mgl32.Vec3{float32(x) * 2, float32(y) * 2, float32(z)*2 - 5}
					size := mgl32.Vec3{1, 1, 1}
					rot := mgl32.QuatRotate(float32(rl.GetTime()), mgl32.Vec3{0, 1, 0})
					color := mgl32.Vec4{float32(x+1) / 2, float32(y+1) / 2, float32(z+1) / 2, 1}

					rend.PushPrimitiveBlock(pos, size, rot, color, "cube")
				}
			}
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
		rend.PushPrimitiveBlock(
			mgl32.Vec3{cam.Position.X(), cam.Position.Y(), cam.Position.Z()}, // position
			mgl32.Vec3{1, 1, 1},    // size
			rot,                    // rotation quaternion
			mgl32.Vec4{1, 0, 0, 1}, // color (red)
			"LightCube",
		)
		rend.PushUIText(
			mgl32.Vec3{0, 10, -5},
			mgl32.Vec4{1, 0, 0, 1},
			fmt.Sprintf("Prims: %d", rend.GetPrimCount()),
		)
		rend.PushUIText(
			mgl32.Vec3{0, 30, -5},
			mgl32.Vec4{1, 0, 0, 1},
			fmt.Sprintf("Light sources: %d", rend.GetLCount()),
		)
		rend.EndFrame(rlCam)

	}
	// Destroy renderer first
}
