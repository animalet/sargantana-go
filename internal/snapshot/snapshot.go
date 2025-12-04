package snapshot

import (
	"github.com/pkg/errors"
	"github.com/tiendc/go-deepcopy"
)

// Copy creates a deep copy of the source object using reflection.
// All slices, maps, and nested pointers are recursively copied to ensure
// true immutability of the returned object.
//
// This function returns an error if the copy operation fails, making it suitable
// for use in contexts where error handling is required (e.g., config loading).
//
// Parameters:
//   - src: Pointer to the object to copy
//
// Returns:
//   - Pointer to a new, deeply-copied object
//   - Error if the copy operation fails
//
// If src is nil, returns (nil, nil).
//
// For constructor use cases where structural errors should never happen,
// consider using MustCopy() instead, which panics on error.
func Copy[T any](src *T) (*T, error) {
	if src == nil {
		return nil, nil
	}

	var dst T
	err := deepcopy.Copy(&dst, &src)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to deep copy type %T", src)
	}

	return &dst, nil
}

// MustCopy creates a deep copy of the source object and panics if the operation fails.
// This is intended for use in constructors and initialization code where configuration
// structures should always be copyable, and a copy failure indicates a programming error.
//
// Use this function to create immutable snapshots at constructor boundaries:
//
//	func NewServer(cfg SargantanaConfig) *Server {
//	    return &Server{
//	        config: *snapshot.MustCopy(&cfg),  // Panic if copy fails
//	    }
//	}
//
// The panic behavior is deliberate: if a configuration struct cannot be deep-copied,
// it indicates a structural problem that should be caught during development, not at runtime.
//
// For contexts where you need to handle copy errors gracefully, use Copy() instead.
//
// If src is nil, returns nil (does not panic).
func MustCopy[T any](src *T) *T {
	if src == nil {
		return nil
	}

	result, err := Copy(src)
	if err != nil {
		// This should never happen with valid config structs.
		// If it does, it indicates a programming error.
		panic("failed to create immutable snapshot: " + err.Error())
	}

	return result
}
