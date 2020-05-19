package pggen

// field_set.go defines a bitset used for specifying subsets of fields

import (
	"github.com/willf/bitset"
)

// A bitset to use to select a subset of fields to update when calling
// a generated Update<Entity> method. FieldSets are reference types like
// slices or maps, so if you want to copy one, use the Clone method.
type FieldSet struct {
	b *bitset.BitSet
}

// Create a new field set with a hint that length bits will be needed.
func NewFieldSet(lengthHint int) FieldSet {
	return FieldSet{b: bitset.New(uint(lengthHint))}
}

// Create a new field set with the first `length` bits set
func NewFieldSetFilled(length int) FieldSet {
	fs := NewFieldSet(length)
	for i := 0; i < length; i++ {
		fs.Set(i, true)
	}
	return fs
}

// Deep copy the field set.
func (fs FieldSet) Clone() FieldSet {
	return FieldSet{b: fs.b.Clone()}
}

// Set the bit at position `bit` to `value`. Can be chained.
func (fs FieldSet) Set(bit int, value bool) FieldSet {
	fs.b.SetTo(uint(bit), value)
	return fs
}

// Return the value of the given bit
func (fs FieldSet) Test(bit int) bool {
	return fs.b.Test(uint(bit))
}

// Return the number of bits set to 1
func (fs FieldSet) CountSetBits() int {
	return int(fs.b.Count())
}
