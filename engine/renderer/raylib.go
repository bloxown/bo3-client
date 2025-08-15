package renderer

import (
	"fmt"

	rl "github.com/gen2brain/raylib-go/raylib"
	"github.com/go-gl/mathgl/mgl32"
)

type Renderer struct {
	width, height int
	queue         []Primitive
	uiqueue       []UIElement
	lights        []Light
	shader        rl.Shader
	cubeModel     rl.Model
}

type Primitive struct {
	Position mgl32.Vec3
	Size     mgl32.Vec3
	Rotation mgl32.Quat
	Color    mgl32.Vec4
	Type     string
}

type UIElement struct {
	Position mgl32.Vec3
	Size     mgl32.Vec3
	Rotation mgl32.Quat
	Color    mgl32.Vec4
	Content  string
	Type     string
}

type Light struct {
	Position  mgl32.Vec3
	Color     mgl32.Vec3
	Intensity float32
	Type      int // 0 = directional, 1 = point, 2 = spot
}

func NewRenderer(width, height int) *Renderer {
	// Load lighting shader with vertex shader too
	shader := rl.LoadShader("lighting.vs", "lighting.fs")

	// Create cube model with proper normals
	cubeMesh := rl.GenMeshCube(1.0, 1.0, 1.0)
	cubeModel := rl.LoadModelFromMesh(cubeMesh)
	cubeModel.Materials.Shader = shader

	return &Renderer{
		width:     width,
		height:    height,
		queue:     []Primitive{},
		uiqueue:   []UIElement{},
		lights:    []Light{},
		shader:    shader,
		cubeModel: cubeModel,
	}
}

func (r *Renderer) ShouldClose() bool {
	return rl.WindowShouldClose()
}

func (r *Renderer) BeginFrame() {
	rl.BeginDrawing()
	rl.ClearBackground(rl.NewColor(51, 26, 26, 255))
	r.queue = r.queue[:0]
	r.uiqueue = r.uiqueue[:0]

}

func (r *Renderer) PushPrimitiveBlock(pos, size mgl32.Vec3, rot mgl32.Quat, color mgl32.Vec4, typetheCube string) {
	r.queue = append(r.queue, Primitive{
		Position: pos,
		Size:     size,
		Rotation: rot,
		Color:    color,
		Type:     typetheCube,
	})
}

func (r *Renderer) PushUIText(pos mgl32.Vec3, color mgl32.Vec4, content string) {
	r.uiqueue = append(r.uiqueue, UIElement{
		Position: pos,
		Color:    color,
		Content:  content,
		Type:     "text",
	})
}

// AddLight adds a light to the scene
func (r *Renderer) AddLight(pos, color mgl32.Vec3, intensity float32, lightType int) {
	r.lights = append(r.lights, Light{
		Position:  pos,
		Color:     color,
		Intensity: intensity,
		Type:      lightType,
	})
}

// AddGlobalLight sets global ambient lighting
func (r *Renderer) AddGlobalLight(color mgl32.Vec3, intensity float32) {
	globalColor := []float32{color.X(), color.Y(), color.Z()}
	globalIntensity := []float32{intensity}

	rl.SetShaderValue(r.shader, rl.GetShaderLocation(r.shader, "globalLightColor"), globalColor, rl.ShaderUniformVec3)
	rl.SetShaderValue(r.shader, rl.GetShaderLocation(r.shader, "globalLightIntensity"), globalIntensity, rl.ShaderUniformFloat)
}

// AddSunLight sets directional sun lighting
func (r *Renderer) AddSunLight(direction, color mgl32.Vec3, intensity float32) {
	sunDir := []float32{direction.X(), direction.Y(), direction.Z()}
	sunColor := []float32{color.X(), color.Y(), color.Z()}
	sunIntensity := []float32{intensity}

	rl.SetShaderValue(r.shader, rl.GetShaderLocation(r.shader, "sunDirection"), sunDir, rl.ShaderUniformVec3)
	rl.SetShaderValue(r.shader, rl.GetShaderLocation(r.shader, "sunColor"), sunColor, rl.ShaderUniformVec3)
	rl.SetShaderValue(r.shader, rl.GetShaderLocation(r.shader, "sunIntensity"), sunIntensity, rl.ShaderUniformFloat)
}

func (r *Renderer) GetPrimCount() int {
	return len(r.queue)
}
func (r *Renderer) GetLCount() int {
	return len(r.lights)
}
func (r *Renderer) GetUICount() int {
	return len(r.uiqueue)
}

// helper to convert mgl32.Vec4 color to Raylib Color
func vec4ToColor(c mgl32.Vec4) rl.Color {
	return rl.NewColor(
		uint8(c[0]*255),
		uint8(c[1]*255),
		uint8(c[2]*255),
		uint8(c[3]*255),
	)
}

func (r *Renderer) EndFrame(rlCam rl.Camera) {
	// Set up lighting uniforms for shader
	rl.BeginShaderMode(r.shader)

	// Pass camera position to shader
	camPos := []float32{rlCam.Position.X, rlCam.Position.Y, rlCam.Position.Z}
	rl.SetShaderValue(r.shader, rl.GetShaderLocation(r.shader, "viewPos"), camPos, rl.ShaderUniformVec3)

	// Pass number of lights
	lightCount := int32(len(r.lights))
	lightCountSlice := []float32{float32(lightCount)}
	rl.SetShaderValue(r.shader, rl.GetShaderLocation(r.shader, "lightCount"), lightCountSlice, rl.ShaderUniformInt)

	// Pass light data (up to 8 lights for performance)
	maxLights := 8
	if len(r.lights) > maxLights {
		r.lights = r.lights[:maxLights]
	}

	for i, light := range r.lights {
		posLoc := rl.GetShaderLocation(r.shader, fmt.Sprintf("lights[%d].position", i))
		colorLoc := rl.GetShaderLocation(r.shader, fmt.Sprintf("lights[%d].color", i))
		intensityLoc := rl.GetShaderLocation(r.shader, fmt.Sprintf("lights[%d].intensity", i))

		pos := []float32{light.Position.X(), light.Position.Y(), light.Position.Z()}
		color := []float32{light.Color.X(), light.Color.Y(), light.Color.Z()}
		intensity := []float32{light.Intensity}

		rl.SetShaderValue(r.shader, posLoc, pos, rl.ShaderUniformVec3)
		rl.SetShaderValue(r.shader, colorLoc, color, rl.ShaderUniformVec3)
		rl.SetShaderValue(r.shader, intensityLoc, intensity, rl.ShaderUniformFloat)
	}
	r.lights = r.lights[:0]
	// Render 3D primitives
	rl.BeginMode3D(rlCam)

	for _, prim := range r.queue {
		col := vec4ToColor(prim.Color)
		switch prim.Type {
		case "cube":
			// Use model instead of DrawCube for proper lighting
			rl.DrawModelEx(r.cubeModel,
				rl.Vector3{X: prim.Position.X(), Y: prim.Position.Y(), Z: prim.Position.Z()},
				rl.Vector3{X: 0, Y: 0, Z: 0}, // rotation axis
				0.0,                          // rotation angle
				rl.Vector3{X: prim.Size.X(), Y: prim.Size.Y(), Z: prim.Size.Z()}, // scale
				col)
		case "LightCube":
			// Use model for light cubes too
			rl.DrawModelEx(r.cubeModel,
				rl.Vector3{X: prim.Position.X(), Y: prim.Position.Y(), Z: prim.Position.Z()},
				rl.Vector3{X: 0, Y: 0, Z: 0}, // rotation axis
				0.0,                          // rotation angle
				rl.Vector3{X: prim.Size.X(), Y: prim.Size.Y(), Z: prim.Size.Z()}, // scale
				col)

			// Add this cube as a light source
			lightColor := mgl32.Vec3{prim.Color.X(), prim.Color.Y(), prim.Color.Z()}
			r.AddLight(prim.Position, lightColor, 1.0, 1) // Point light with intensity 1.0
		}
	}

	rl.EndMode3D()
	rl.EndShaderMode()

	// Render UI elements (no lighting needed)
	for _, ui := range r.uiqueue {
		switch ui.Type {
		case "text":
			rl.DrawText(ui.Content, int32(ui.Position.X()), int32(ui.Position.Y()), 20, vec4ToColor(ui.Color))
		}
	}

	rl.EndDrawing()

	// clear queues for next frame
	r.queue = r.queue[:0]
	r.uiqueue = r.uiqueue[:0]
}

func (r *Renderer) Destroy() {
	rl.UnloadModel(r.cubeModel)
	rl.UnloadShader(r.shader)
	rl.CloseWindow()
}
