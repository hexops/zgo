package errors

import (
	"errors"
	"fmt"
)

// Wrap returns an error annotating err with the specified message. If err is nil, Wrap returns nil.
func Wrap(err error, msg string) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%v: %v", msg, err)
}

// Wrapf returns an error annotating err with the specified message. If err is nil, Wrap returns nil.
func Wrapf(err error, format string, args ...any) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf(format+": %v", append(args, err))
}

var (
	// New returns an error that formats as the given text.
	// Each call to New returns a distinct error value even if the text is identical.
	//
	// func New(text string) error
	New = errors.New

	// Join returns an error that wraps the given errors.
	// Any nil error values are discarded.
	// Join returns nil if errs contains no non-nil values.
	// The error formats as the concatenation of the strings obtained
	// by calling the Error method of each element of errs, with a newline
	// between each string.
	//
	// func Join(errs ...error) error
	Join = errors.Join

	// Unwrap returns the result of calling the Unwrap method on err, if err's
	// type contains an Unwrap method returning error.
	// Otherwise, Unwrap returns nil.
	//
	// Unwrap returns nil if the Unwrap method returns []error.
	//
	// func Unwrap(err error) error
	Unwrap = errors.Unwrap

	// Is reports whether any error in err's tree matches target.
	//
	// The tree consists of err itself, followed by the errors obtained by repeatedly
	// calling Unwrap. When err wraps multiple errors, Is examines err followed by a
	// depth-first traversal of its children.
	//
	// An error is considered to match a target if it is equal to that target or if
	// it implements a method Is(error) bool such that Is(target) returns true.
	//
	// An error type might provide an Is method so it can be treated as equivalent
	// to an existing error. For example, if MyError defines
	//
	//	func (m MyError) Is(target error) bool { return target == fs.ErrExist }
	//
	// then Is(MyError{}, fs.ErrExist) returns true. See syscall.Errno.Is for
	// an example in the standard library. An Is method should only shallowly
	// compare err and the target and not call Unwrap on either.
	//
	// func Is(err, target error) bool
	Is = errors.Is

	// As finds the first error in err's tree that matches target, and if one is found, sets
	// target to that error value and returns true. Otherwise, it returns false.
	//
	// The tree consists of err itself, followed by the errors obtained by repeatedly
	// calling Unwrap. When err wraps multiple errors, As examines err followed by a
	// depth-first traversal of its children.
	//
	// An error matches target if the error's concrete value is assignable to the value
	// pointed to by target, or if the error has a method As(interface{}) bool such that
	// As(target) returns true. In the latter case, the As method is responsible for
	// setting target.
	//
	// An error type might provide an As method so it can be treated as if it were a
	// different error type.
	//
	// As panics if target is not a non-nil pointer to either a type that implements
	// error, or to any interface type.
	//
	// func As(err error, target any) bool
	As = errors.As
)
