package instance

import (
	"context"
)

type Redis interface {
	Ping(ctx context.Context) error
	Subscribe(ctx context.Context, ch chan string, subscribeTo ...string)
}
