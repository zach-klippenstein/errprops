# errprops [![GoDoc](https://godoc.org/github.com/zach-klippenstein/errprops?status.svg)](https://godoc.org/github.com/zach-klippenstein/errprops)

errprops provides a fluent API for setting and querying key/value pairs on errors.

It is compatible with [Dave Cheney's github.com/pkg/errors](https://godoc.org/github.com/pkg/errors) package:
errors returned by this package support both `Stacktrace()` and `Cause()`.
They also support `fmt.Formatter` – key/value pairs are formatted using the format
flags passed to `Format`, and the wrapped error is asked to format itself.

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

See the Examples in the godoc for more.
