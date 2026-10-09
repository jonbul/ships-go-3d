package game

import (
	"math"

	"ships3d/models"
)

type vec3 = models.Vec3

// quat is a rotation quaternion as [x, y, z, w], three.js' component order.
type quat [4]float64

var identityQuat = quat{0, 0, 0, 1}

func add(a, b vec3) vec3         { return vec3{a[0] + b[0], a[1] + b[1], a[2] + b[2]} }
func sub(a, b vec3) vec3         { return vec3{a[0] - b[0], a[1] - b[1], a[2] - b[2]} }
func mul(a vec3, s float64) vec3 { return vec3{a[0] * s, a[1] * s, a[2] * s} }
func dot(a, b vec3) float64      { return a[0]*b[0] + a[1]*b[1] + a[2]*b[2] }
func length(a vec3) float64      { return math.Sqrt(dot(a, a)) }

func finiteVec(a vec3) bool {
	for _, c := range a {
		if math.IsNaN(c) || math.IsInf(c, 0) {
			return false
		}
	}
	return true
}

func normalize(a vec3) (vec3, bool) {
	l := length(a)
	if l < 1e-9 || !finiteVec(a) {
		return vec3{}, false
	}
	return mul(a, 1/l), true
}

// clampLength shortens a to at most max, keeping its direction.
func clampLength(a vec3, max float64) vec3 {
	l := length(a)
	if l <= max {
		return a
	}
	return mul(a, max/l)
}

func normalizeQuat(q quat) (quat, bool) {
	l := math.Sqrt(q[0]*q[0] + q[1]*q[1] + q[2]*q[2] + q[3]*q[3])
	if l < 1e-9 || math.IsNaN(l) || math.IsInf(l, 0) {
		return identityQuat, false
	}
	return quat{q[0] / l, q[1] / l, q[2] / l, q[3] / l}, true
}

// yawQuat is a rotation of angle radians around +Y.
func yawQuat(angle float64) quat {
	return quat{0, math.Sin(angle / 2), 0, math.Cos(angle / 2)}
}

// distanceToSegment is the distance from point p to the segment a-b.
func distanceToSegment(p, a, b vec3) float64 {
	ab := sub(b, a)
	lengthSq := dot(ab, ab)
	if lengthSq < 1e-12 {
		return length(sub(p, a))
	}
	t := math.Max(0, math.Min(1, dot(sub(p, a), ab)/lengthSq))
	return length(sub(p, add(a, mul(ab, t))))
}
