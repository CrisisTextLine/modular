// Package edwards25519 provides a stub implementation of the edwards25519
// group operations. This stub replaces filippo.io/edwards25519, which is only
// needed for MariaDB's client_ed25519 authentication plugin — a feature not
// used by this project. All operations return errors to ensure the
// unsupported auth path is never silently exercised.
//
// The full implementation is available at filippo.io/edwards25519 for
// projects that require MariaDB ed25519 authentication.
package edwards25519

import "errors"

// errNotSupported is returned by all stub operations.
var errNotSupported = errors.New("edwards25519: MariaDB client_ed25519 auth is not supported; use a different authentication plugin")

// Scalar represents an integer modulo the prime order of the edwards25519 group.
// This is a stub type; all operations return errors.
type Scalar struct{}

// Point represents a point on the edwards25519 curve.
// This is a stub type; all operations return errors.
type Point struct{}

// NewScalar returns a new zero Scalar.
func NewScalar() *Scalar {
	return &Scalar{}
}

// NewIdentityPoint returns the identity point.
func NewIdentityPoint() *Point {
	return &Point{}
}

// NewGeneratorPoint returns the canonical generator point.
func NewGeneratorPoint() *Point {
	return &Point{}
}

// SetBytesWithClamping is not supported and returns an error.
// This causes the MariaDB client_ed25519 auth path to fail gracefully.
func (s *Scalar) SetBytesWithClamping(x []byte) (*Scalar, error) {
	return nil, errNotSupported
}

// SetUniformBytes is not supported and returns an error.
func (s *Scalar) SetUniformBytes(x []byte) (*Scalar, error) {
	return nil, errNotSupported
}

// SetCanonicalBytes is not supported and returns an error.
func (s *Scalar) SetCanonicalBytes(x []byte) (*Scalar, error) {
	return nil, errNotSupported
}

// Bytes returns a zero-value encoding. Only reached if prior operations
// erroneously succeeded; the stub ensures they do not.
func (s *Scalar) Bytes() []byte {
	return make([]byte, 32)
}

// MultiplyAdd sets v = x*y+z and returns v.
func (v *Scalar) MultiplyAdd(x, y, z *Scalar) *Scalar {
	return v
}

// Add sets v = x+y and returns v.
func (v *Scalar) Add(x, y *Scalar) *Scalar {
	return v
}

// Subtract sets v = x-y and returns v.
func (v *Scalar) Subtract(x, y *Scalar) *Scalar {
	return v
}

// Negate sets v = -x and returns v.
func (v *Scalar) Negate(x *Scalar) *Scalar {
	return v
}

// Multiply sets v = x*y and returns v.
func (v *Scalar) Multiply(x, y *Scalar) *Scalar {
	return v
}

// Set sets v = x and returns v.
func (v *Scalar) Set(x *Scalar) *Scalar {
	return v
}

// Equal returns 1 if v and t are equal, 0 otherwise.
func (v *Scalar) Equal(t *Scalar) int {
	return 0
}

// ScalarBaseMult sets v = x*B and returns v.
func (v *Point) ScalarBaseMult(x *Scalar) *Point {
	return v
}

// VarTimeDoubleScalarBaseMult sets v = a*A + b*B and returns v.
func (v *Point) VarTimeDoubleScalarBaseMult(a *Scalar, A *Point, b *Scalar) *Point {
	return v
}

// Add sets v = p+q and returns v.
func (v *Point) Add(p, q *Point) *Point {
	return v
}

// Subtract sets v = p-q and returns v.
func (v *Point) Subtract(p, q *Point) *Point {
	return v
}

// Negate sets v = -p and returns v.
func (v *Point) Negate(p *Point) *Point {
	return v
}

// Set sets v = u and returns v.
func (v *Point) Set(u *Point) *Point {
	return v
}

// SetBytes is not supported and returns an error.
func (v *Point) SetBytes(x []byte) (*Point, error) {
	return nil, errNotSupported
}

// Bytes returns a zero-value encoding.
func (v *Point) Bytes() []byte {
	return make([]byte, 32)
}

// Equal returns 1 if v and u represent the same point, 0 otherwise.
func (v *Point) Equal(u *Point) int {
	return 0
}
