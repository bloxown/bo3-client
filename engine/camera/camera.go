package camera

import (
	"math"

	"github.com/go-gl/mathgl/mgl32"
)

// Camera is a simple freecam camera.
type Camera struct {
	Position mgl32.Vec3
	Front    mgl32.Vec3
	Up       mgl32.Vec3
	Right    mgl32.Vec3
	WorldUp  mgl32.Vec3

	Yaw   float32
	Pitch float32

	Speed       float32
	Sensitivity float32

	// projection params
	FOV    float32
	Aspect float32
	Near   float32
	Far    float32
}

// NewCamera creates a camera positioned at pos, looking with yaw/pitch (degrees).
// up is usually mgl32.Vec3{0,1,0}.
func NewCamera(pos, up mgl32.Vec3, yaw, pitch float32) *Camera {
	c := &Camera{
		Position:    pos,
		WorldUp:     up,
		Yaw:         yaw,
		Pitch:       pitch,
		Speed:       5.0,
		Sensitivity: 0.1,
		// sensible defaults for projection; call SetAspect() to tune aspect ratio
		FOV:    45.0,
		Aspect: 4.0 / 3.0,
		Near:   0.1,
		Far:    100.0,
	}
	c.updateCameraVectors()
	return c
}

// SetAspect updates the projection aspect ratio (call on window resize).
func (c *Camera) SetAspect(aspect float32) {
	c.Aspect = aspect
}

// ProcessKeyboard moves the camera using WASD booleans and delta time (seconds).
func (c *Camera) ProcessKeyboard(forward, backward, left, right bool, deltaTime float32) {
	velocity := c.Speed * deltaTime
	if forward {
		c.Position = c.Position.Add(c.Front.Mul(velocity))
	}
	if backward {
		c.Position = c.Position.Sub(c.Front.Mul(velocity))
	}
	if left {
		c.Position = c.Position.Sub(c.Right.Mul(velocity))
	}
	if right {
		c.Position = c.Position.Add(c.Right.Mul(velocity))
	}
}

// ProcessMouse adjusts yaw/pitch from mouse delta (dx,dy) in pixels.
// Use small sensitivity for sane rotation.
func (c *Camera) ProcessMouse(dx, dy float32) {
	c.Yaw += dx * c.Sensitivity
	c.Pitch -= dy * c.Sensitivity

	if c.Pitch > 89.0 {
		c.Pitch = 89.0
	}
	if c.Pitch < -89.0 {
		c.Pitch = -89.0
	}

	c.updateCameraVectors()
	//log.Printf("Yaw=%.2f Pitch=%.2f Front=%v\n", c.Yaw, c.Pitch, c.Front)

}

// GetViewMatrix returns the view matrix (mgl32.Mat4) for the current camera transform.
func (c *Camera) GetViewMatrix() mgl32.Mat4 {
	target := c.Position.Add(c.Front)
	return mgl32.LookAtV(c.Position, target, c.Up)
}

// GetProjectionMatrix returns a perspective projection matrix (mgl32.Mat4).
// Uses current FOV (degrees), Aspect, Near and Far.
func (c *Camera) GetProjectionMatrix() mgl32.Mat4 {
	// math.Pi is float64, so do the fovy calc in float64 then cast to float32
	fovyRad := float32(float64(c.FOV) * math.Pi / 180.0)
	return mgl32.Perspective(fovyRad, c.Aspect, c.Near, c.Far)
}

// internal: recompute front/right/up vectors from yaw/pitch
func (c *Camera) updateCameraVectors() {
	// Convert degrees to radians in float64 for math trig functions
	yawRad := float64(c.Yaw) * math.Pi / 180.0
	pitchRad := float64(c.Pitch) * math.Pi / 180.0

	// compute using float64, then cast to float32
	fx := float32(math.Cos(yawRad) * math.Cos(pitchRad))
	fy := float32(math.Sin(pitchRad))
	fz := float32(math.Sin(yawRad) * math.Cos(pitchRad))

	front := mgl32.Vec3{fx, fy, fz}.Normalize()

	c.Front = front
	c.Right = front.Cross(c.WorldUp).Normalize()
	c.Up = c.Right.Cross(c.Front).Normalize()
}
