package vm

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// What a thing weighs, which is not what its own weight says.
//
// A container weighs itself plus what is in it, and the contents of a
// container are the objects whose location is the container's own number —
// that is the format's trick, and the reason an object number and a location
// share one byte.
//
// Then there is the rule nobody would guess, which the manual spends a whole
// sentence on: a container of zero weight transmits zero weight for its
// contents as well as for itself. It is the magic bag, and without it an
// adventure that hands the player a weightless sack finds the sack refusing
// to be picked up once there is anything in it.
func TestWhatAThingWeighs(t *testing.T) {
	// A sack holding a rock and a feather, and a weightless bag holding an
	// anvil. Both bags sit in a room, so neither is in the other and neither
	// is carried; object 3 is loose somewhere else so that nothing sums it.
	//
	// The room numbers are deliberately far from any object number here. The
	// two share a byte, so a bag left at location 0 would be a bag inside
	// object 0 — which is the thing TestAContainerThatHoldsItselfDoesNotRunForever
	// is for, and has no business happening by accident in this one.
	const (
		room  = 50
		other = 60
	)

	m := &Machine{objects: []object{
		{weight: 2, container: true, location: room}, // 0: the sack
		{weight: 10, location: 0},                    // 1: a rock, in the sack
		{weight: 1, location: 0},                     // 2: a feather, in the sack
		{weight: 7, location: other},                 // 3: somewhere else entirely
		{weight: 0, container: true, location: room}, // 4: the magic bag
		{weight: 200, location: 4},                   // 5: an anvil, in the magic bag
		{weight: 3, location: LocCarried},            // 6: held
		{weight: 4, location: LocWorn},               // 7: worn
	}}

	require.Equal(t, 13, m.objectWeight(0),
		"the sack is itself plus the rock and the feather")
	require.Equal(t, 10, m.objectWeight(1), "a rock is a rock")
	require.Equal(t, 0, m.objectWeight(4),
		"a container of zero weight transmits zero, anvil and all")
	require.Equal(t, 0, m.objectWeight(99), "an object that is not there weighs nothing")

	require.Equal(t, 7, m.weightCarried(),
		"what is held and what is worn, and nothing that is merely lying about")
}

// And the total stays inside the byte a flag holds.
//
// The manual says the maximum "will not be exceeded" rather than that it
// wraps, which matters: a weight that wrapped would read as light, and the
// strength check would wave through exactly the load it exists to refuse.
func TestWeightNeverWraps(t *testing.T) {
	m := &Machine{objects: []object{
		{weight: 200, location: LocCarried},
		{weight: 200, location: LocCarried},
		{weight: 200, location: LocWorn},
	}}

	require.Equal(t, 255, m.weightCarried())
}

// A container holding itself is bounded rather than fatal.
//
// No author writes one. A half-built database or a bad INSERT can hold one
// easily, and the reference descends without a bound — so the ten levels the
// manual names are not only fidelity, they are what stands between a strange
// object and a stack overflow.
func TestAContainerThatHoldsItselfDoesNotRunForever(t *testing.T) {
	m := &Machine{objects: []object{
		{weight: 1, container: true, location: 0}, // 0 is inside 0
	}}

	require.LessOrEqual(t, m.objectWeight(0), 255,
		"it came back, which is the whole of what this test asks")
}
