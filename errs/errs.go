// Package errs can control telegram error handler behavior
package errs

type Error struct {
	Message    string
	NotifyUser bool
	PrintError bool
}

func (s *Error) Error() string {
	return s.Message
}
