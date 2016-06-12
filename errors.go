/*
Package errprops provides a fluent API for setting and querying key/value pairs on errors.

It is compatible with Dave Cheney's github.com/pkg/errors package:
errors returned by this package support both Stacktrace() and Cause().
They also support fmt.Formatter – key/value pairs are formatted using the format
flags passed to Format, and the wrapped error is asked to format itself.

A (contrived) example:
	func DoThing() error {
		return errors.New("fail")
	}

	func DoThingWithContext(id int) error {
		if err := DoThing(); err != nil {
			return errprops.From(err).WithValue("id", id)
		}
		return nil
	}

	func main() {
		id := 42
		if err := DoThingWithContext(id); err != nil {
			log.Printf("%d: %+v", errprops.GetOptional(err, "id"), err)
		}
	}

This would output something like:
	42: [id=42] fail
	github.com/path/to/package.DoThing
		/local/path/to/package/main.go:2
	github.com/path/to/package.DoThingWithContext
		/local/path/to/package/main.go:6
	github.com/path/to/package.main
		/local/path/to/package/main.go:14

See the Examples for more.
*/
package errprops

import (
	"fmt"

	"github.com/pkg/errors"
)

// Specified by github.com/pkg/errors
type hasCause interface {
	Cause() error
}

// Specified by github.com/pkg/errors
type hasStacktrace interface {
	Stacktrace() errors.Stacktrace
}

type hasProps interface {
	// Returns the value associated with the given key on the current error.
	// Should *NOT* recurse into the error's cause, if it has one.
	Get(key interface{}) (value interface{}, ok bool)
}

// An implementation of the standard error interface that can set key/value pairs.
// PropErrors are immutable.
//
// This interface should never be used as a return *type* for a function.
//
// Good:
//		func do() error { return errprops.From(…) }
// Bad:
//		func do() PropError { return errprops.From(…) }
//
type PropError interface {
	error
	hasCause
	hasStacktrace
	hasProps
	fmt.Formatter

	// Returns a copy of this PropError with the specified key/value pair.
	// Does not modify the current PropError.
	WithValue(key, value interface{}) PropError

	// If the wrapped error implements fmt.Formatter, this method should delegate directly
	// to it.
	formatBaseError(f fmt.State, c rune)
}

// Returns a PropError that can be used to set properties on err.
// The original err is not modified.
//
// Intended to be used in a fluent style, like:
// 		return From(err).
//			WithValue("someKey", someValue).
// 			WithValue("otherKey", otherValue)
func From(err error) PropError {
	return baseError{err}
}

// Get returns the value associated with key using the following rules to resolve key:
//   - If the key exists on the current error, return the value set by the last call to WithValue.
//   - Else, if the current error has a cause, recursively look for the key on the cause.
func Get(err error, key interface{}) (interface{}, bool) {
	if err == nil {
		return nil, false
	}

	if err, ok := err.(hasProps); ok {
		if val, ok := err.Get(key); ok {
			return val, true
		}
	}

	if err, ok := err.(hasCause); ok {
		return Get(err.Cause(), key)
	}

	return nil, false
}

// GetOptional is the same as Get but just returns nil if the key isn't set, to make it
// more convenient to use in single-value contexts.
func GetOptional(err error, key interface{}) interface{} {
	if val, ok := Get(err, key); ok {
		return val
	}
	return nil
}

// Implementation of PropError that just delegates most calls to the underlying error
// if it supports them.
type baseError struct {
	error
}

func (e baseError) Cause() error {
	if e, ok := e.error.(hasCause); ok {
		return e.Cause()
	}
	return nil
}

func (e baseError) Stacktrace() errors.Stacktrace {
	if e, ok := e.error.(hasStacktrace); ok {
		return e.Stacktrace()
	}
	return nil
}

func (e baseError) Get(key interface{}) (interface{}, bool) {
	if e, ok := e.error.(hasProps); ok {
		return e.Get(key)
	}
	return nil, false
}

func (e baseError) WithValue(key, value interface{}) PropError {
	return &keyValueError{e, key, value}
}

func (e baseError) Format(f fmt.State, c rune) {
	e.formatBaseError(f, c)
}

func (e baseError) formatBaseError(f fmt.State, c rune) {
	if e, ok := e.error.(fmt.Formatter); ok {
		e.Format(f, c)
		return
	}
	fmt.Fprint(f, e.error)
}

// A wrapper around an Error that associates a key with a value.
type keyValueError struct {
	PropError
	key, value interface{}
}

func (e *keyValueError) Get(key interface{}) (interface{}, bool) {
	if key == e.key {
		return e.value, true
	}
	return e.PropError.Get(key)
}

func (e *keyValueError) WithValue(key, value interface{}) PropError {
	// Override here so the wrapped error is this object, not the embedded PropError.
	return &keyValueError{e, key, value}
}

func (e *keyValueError) formatBaseError(f fmt.State, c rune) {
	e.PropError.formatBaseError(f, c)
}

func (e *keyValueError) Format(f fmt.State, c rune) {
	fmt.Fprint(f, "[")
	e.formatInner(f, c)
	fmt.Fprint(f, "] ")
	e.formatBaseError(f, c)
}

func (e *keyValueError) formatInner(f fmt.State, c rune) {
	format := "%"

	if f.Flag('+') {
		format += "+"
	} else if f.Flag('#') {
		format += "#"
	}
	format = string(append([]rune(format), c))

	fmt.Fprintf(f, format+"="+format, e.key, e.value)

	if e, ok := e.PropError.(*keyValueError); ok {
		fmt.Fprint(f, ",")
		e.formatInner(f, c)
	}
}
