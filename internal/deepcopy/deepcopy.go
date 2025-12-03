package deepcopy

import (
	"github.com/jinzhu/copier"
	"github.com/pkg/errors"
)

// Copy creates a deep copy of the source object using reflection.
// All slices, maps, and nested pointers are recursively copied to ensure
// true immutability of the returned object.
//
// Parameters:
//   - src: Pointer to the object to copy
//
// Returns:
//   - Pointer to a new, deeply-copied object
//   - Error if the copy operation fails
//
// If src is nil, returns (nil, nil).
func Copy[T any](src *T) (*T, error) {
	if src == nil {
		return nil, nil
	}

	var dst T
	err := copier.CopyWithOption(&dst, src, copier.Option{
		IgnoreEmpty: false, // Copy zero values
		DeepCopy:    true,  // Recursively copy slices, maps, nested structs
	})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to deep copy config of type %T", src)
	}

	return &dst, nil
}
