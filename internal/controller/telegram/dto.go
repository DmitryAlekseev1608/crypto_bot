package telegram

import (
	"context"
	"time"
)

type clientUpdate struct {
	cancelFunc context.CancelFunc
	time       time.Time
}
