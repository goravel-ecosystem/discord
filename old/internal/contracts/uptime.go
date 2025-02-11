package contracts

import (
	"context"
)

type Uptime interface {
	// Monitor starts monitoring a website and sends updates to Discord.
	Monitor(ctx context.Context)
}
