package domain

import "errors"

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
