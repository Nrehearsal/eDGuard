package pump

import "context"

type Interface interface {
	Start(ctx context.Context) error
	PopOne()
}
