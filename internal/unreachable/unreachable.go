package unreachable

import "fmt"

const _msg = "%v\n\nThis is a bug in go.uber.org/config - there's" +
	"nothing you can do to fix it. Please file an issue at" +
	"https://github.com/uber-go/config/issues/new. Sorry!"

// Wrap clarifies that an error's root cause is a bug in the config library
// and directs the user to file a bug report.
func Wrap(err error) error {
	return fmt.Errorf(_msg, err)
}
