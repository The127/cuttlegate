package domain

import (
	"errors"
	"fmt"
)

// ValidationError is returned when a domain entity fails its invariants.
// Field names the invalid field; Message describes the constraint violation.
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("%s: %s", e.Field, e.Message)
}

// ErrNotFound is returned when a requested entity does not exist.
var ErrNotFound = errors.New("not found")

// ErrConflict is returned when an operation would violate a uniqueness constraint.
var ErrConflict = errors.New("conflict")

// ErrLastAdmin is returned when removing a member would leave a project with no admin.
var ErrLastAdmin = errors.New("last admin")

// ErrImmutableVariants is returned when a variant mutation is attempted on a bool flag.
var ErrImmutableVariants = errors.New("bool flag variants are immutable")

// ErrDefaultVariant is returned when attempting to delete the default variant key.
var ErrDefaultVariant = errors.New("cannot delete the default variant")

// ErrLastVariant is returned when deleting a variant would leave a flag with no variants.
var ErrLastVariant = errors.New("cannot remove the last variant")

// ErrPriorityConflict is returned when a rule's priority collides with an existing rule in the same flag+environment.
var ErrPriorityConflict = errors.New("a rule with this priority already exists")

// ErrKeyRevoked is returned when an operation is attempted on a revoked API key.
var ErrKeyRevoked = errors.New("key revoked")

// ErrVariantInUse is returned when deleting a variant that is referenced by targeting rules.
var ErrVariantInUse = errors.New("variant is referenced by targeting rules")
