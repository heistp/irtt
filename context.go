package irtt

import "context"

func isContextError(err error) bool {
	return err == context.Canceled || err == context.DeadlineExceeded
}
