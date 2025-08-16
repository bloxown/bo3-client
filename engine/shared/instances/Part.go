package instances

import (
	"math"

	"github.com/go-gl/mathgl/mgl32"
)

// Part is a simple subclass that acts similar to Roblox's Workspace object; the renderer only renders from here
type Part struct {
	BaseInstance  // embed to inherit behaviour
	Position      mgl32.Vec3
	Size          mgl32.Vec3
	Rot           mgl32.Vec3
	PrimitiveType string
}

// NewPart returns a ready-to-use *Part wrapped as Instance.
func NewPart() Instance {
	f := &Part{}
	// initialize embedded BaseInstance fields explicitly
	f.name = "ExamplePart"
	f.className = "Part"
	f.children = make([]Instance, 0)
	f.PrimitiveType = "Cube"
	f.Size = mgl32.Vec3{1, 1, 1}
	f.Rot = mgl32.Vec3{0, 0, 0}
	f.Position = mgl32.Vec3{0, 0, 0}
	f.this = f
	return f
}

func Vec3ToQuatAxisAngleDegrees(v mgl32.Vec3) mgl32.Quat {
	angleDegrees := v.Len() // vector length = angle in degrees
	if angleDegrees == 0 {
		return mgl32.QuatIdent() // no rotation
	}
	angleRadians := angleDegrees * float32(math.Pi/180.0)
	axis := v.Normalize()
	return mgl32.QuatRotate(angleRadians, axis)
}

// Optionally add Part-specific methods:
func (f *Part) SetPosition(pos mgl32.Vec3) {
	f.Position = pos
}

func (f *Part) GetPosition() mgl32.Vec3 {
	return f.Position
}
func (f *Part) SetSize(s mgl32.Vec3) {
	f.Size = s
}

func (f *Part) GetSize() mgl32.Vec3 {
	return f.Size
}
func (f *Part) SetRot(s mgl32.Vec3) {
	f.Rot = s
}

func (f *Part) GetRot() mgl32.Vec3 {
	return f.Rot
}
func (f *Part) GetType(typ string) {
	f.PrimitiveType = typ
}

func (f *Part) SetType() string {
	return f.PrimitiveType
}
func (f *Part) GetRotRender() mgl32.Quat {
	return Vec3ToQuatAxisAngleDegrees(f.Rot)
}
