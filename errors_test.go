package errprops

import (
	"fmt"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

func Example() {
	var err error = errors.New("fail")
	var errWithProp error = From(err).
		WithValue("key", "value").
		WithValue("foo", "bar")

	fmt.Println("key:", GetOptional(errWithProp, "key"))
	fmt.Println("foo:", GetOptional(errWithProp, "foo"))

	// Output:
	// key: value
	// foo: bar
}

func Example_cause() {
	var rootCause error = From(errors.New("root cause")).
		WithValue("rootKey", "rootValue")

	var wrapped error = From(errors.Wrap(rootCause, "wrapped")).
		WithValue("wrappedKey", "wrappedValue")

	fmt.Println("rootKey:", GetOptional(wrapped, "rootKey"))
	fmt.Println("wrappedKey:", GetOptional(wrapped, "wrappedKey"))

	// Output:
	// rootKey: rootValue
	// wrappedKey: wrappedValue
}

func Example_overriding() {
	var rootCause error = From(errors.New("root cause")).
		WithValue("key", "rootValue")

	var wrapped error = From(errors.Wrap(rootCause, "wrapped")).
		WithValue("key", "wrappedValue")

	fmt.Println("key:", GetOptional(rootCause, "key"))
	fmt.Println("key:", GetOptional(wrapped, "key"))

	// Output:
	// key: rootValue
	// key: wrappedValue
}

func Example_format() {
	type state struct {
		Url       string
		StartTime time.Time
	}

	err := From(io.EOF).
		WithValue("filename", "/tmp/stuff").
		WithValue("state", state{
		"https://example.com",
		time.Date(2016, 06, 12, 9, 47, 16, 0, time.UTC),
	})

	fmt.Printf("%v\n", err)
	fmt.Printf("%+v\n", err)
	fmt.Printf("%#v\n", err)

	// Output:
	// [state={https://example.com 2016-06-12 09:47:16 +0000 UTC},filename=/tmp/stuff] EOF
	// [state={Url:https://example.com StartTime:2016-06-12 09:47:16 +0000 UTC},filename=/tmp/stuff] EOF
	// ["state"=errprops.state{Url:"https://example.com", StartTime:time.Time{sec:63601321636, nsec:0, loc:(*time.Location)(0x5c82e0)}},"filename"="/tmp/stuff"] EOF
}

func TestFrom(t *testing.T) {
	err := From(errors.New("hello"))

	assert.EqualError(t, err, "hello")

	assert.Nil(t, err.Cause())
	assert.Nil(t, errors.Cause(err))

	stacktrace := err.Stacktrace()
	assert.NotNil(t, stacktrace)
	assert.NotEmpty(t, stacktrace)

	nothing, ok := Get(err, nil)
	assert.False(t, ok)
	assert.Nil(t, nothing)

	nothing, ok = Get(err, "nope")
	assert.False(t, ok)
	assert.Nil(t, nothing)
}

func TestCause(t *testing.T) {
	cause := errors.New("cause")
	middle := errors.Wrap(cause, "middle")
	outer := errors.Wrap(middle, "outer")
	err := From(outer)

	assert.EqualError(t, err, "outer: middle: cause")

	assert.Equal(t, middle, err.Cause())
	assert.Equal(t, cause, errors.Cause(err))
}

func TestFormatNoProps(t *testing.T) {
	cause := errors.New("cause")
	err := From(cause)

	assert.Equal(t, "cause", fmt.Sprintf("%v", err))

	formatted := fmt.Sprintf("%+v", err)
	prefix := `cause
github.com/zach-klippenstein/errprops.TestFormatNoProps
	` // The rest of the string will have device-specific components (paths, CPU).

	assert.True(t, strings.HasPrefix(formatted, prefix),
		"expected\n%s\nto have prefix\n%s", formatted, prefix)
	assert.Contains(t, formatted, "github.com/zach-klippenstein/errprops/errors_test.go:")
}

func TestFormatWithProps(t *testing.T) {
	err := From(errors.New("cause")).
		WithValue(struct{ Id int }{1}, 2).
		WithValue("key", "value")

	assert.Equal(t, "[key=value,{1}=2] cause", fmt.Sprintf("%v", err))

	formatted := fmt.Sprintf("%+v", err)
	prefix := `[key=value,{Id:1}=2] cause
github.com/zach-klippenstein/errprops.TestFormatWithProps
	` // The rest of the string will have device-specific components (paths, CPU).

	assert.True(t, strings.HasPrefix(formatted, prefix),
		"expected\n%s\nto have prefix\n%s", formatted, prefix)
	assert.Contains(t, formatted, "github.com/zach-klippenstein/errprops/errors_test.go:")
}

func TestWithValue(t *testing.T) {
	err := From(errors.New("hello"))

	withFooBar := err.WithValue("foo", "bar")
	assertHasPropOnSelf(t, withFooBar, "foo", "bar")
	// Get should only check keys, not values.
	assertDoesNotHaveProp(t, withFooBar, "bar")

	// Change the value.
	withFooBaz := withFooBar.WithValue("foo", "baz")
	assertHasPropOnSelf(t, withFooBaz, "foo", "baz")
	// Original error should remain unchanged.
	assertHasPropOnSelf(t, withFooBar, "foo", "bar")

	// Add another value.
	withFooBazKeyValue := withFooBaz.WithValue("key", "value")
	assertHasPropOnSelf(t, withFooBazKeyValue, "key", "value")
	assertHasPropOnSelf(t, withFooBaz, "foo", "baz")
}

func TestGetIntermediateCauseHasProp(t *testing.T) {
	cause := errors.New("root cause")
	middle := From(errors.Wrap(cause, "")).
		WithValue("foo", "bar")
	err := errors.Wrap(middle, "")

	assertHasProp(t, err, "foo", "bar")

	// Wrapping with another PropError shouldn't change the results.
	err = From(err)
	assertDoesNotHavePropOnSelf(t, err.(PropError), "foo")
	assertHasProp(t, err, "foo", "bar")
}

func TestGetRootCauseHasProp(t *testing.T) {
	cause := From(errors.New("root cause")).
		WithValue("foo", "bar")
	middle := errors.Wrap(cause, "")
	err := errors.Wrap(middle, "")

	assertHasProp(t, err, "foo", "bar")

	// Wrapping with another PropError shouldn't change the results.
	err = From(err)
	assertDoesNotHavePropOnSelf(t, err.(PropError), "foo")
	assertHasProp(t, err, "foo", "bar")
}

func assertHasPropOnSelf(t *testing.T, err PropError, key, wantVal interface{}) {
	val, ok := err.Get(key)
	assert.True(t, ok)
	assert.Equal(t, wantVal, val)
	assertHasProp(t, err, key, wantVal)
}

// Like assertHasPropOnSelf but calls the package Get to recurse through causes.
func assertHasProp(t *testing.T, err error, key, wantVal interface{}) {
	assert.Equal(t, wantVal, GetOptional(err, key))
	val, ok := Get(err, key)
	assert.True(t, ok)
	assert.Equal(t, wantVal, val)
}

func assertDoesNotHavePropOnSelf(t *testing.T, err PropError, key interface{}) {
	val, ok := err.Get(key)
	assert.False(t, ok)
	assert.Nil(t, val)
}

// Like assertDoesNotHavePropOnSelf but calls the package Get to recurse through causes.
func assertDoesNotHaveProp(t *testing.T, err error, key interface{}) {
	assert.Nil(t, GetOptional(err, key))
	val, ok := Get(err, key)
	assert.False(t, ok)
	assert.Nil(t, val)

	if err, ok := err.(PropError); ok {
		assertDoesNotHavePropOnSelf(t, err, key)
	}
}
