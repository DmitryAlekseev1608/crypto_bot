package telegram

import (
	"context"
	"time"
)

type clientUpdate struct {
	cancelF context.CancelFunc
	time    time.Time
}
