package svrctrl

import (
	"context"
	"time"

	"github.com/qumogu/go-tools/logger"
	"golang.org/x/sync/errgroup"
)

func GoAndRestartOnError(ctx context.Context, errGroup *errgroup.Group, name string, f func() error) {
	errGroup.Go(func() error { return RunAndRestartOnError(ctx, name, f) })
}

// RunAndRestartOnError runs function until context done. Always restart if failed.
func RunAndRestartOnError(ctx context.Context, name string, f func() error) error {
	for {
		logger.Infof("starting %s", name)

		err := f()
		if err != nil {
			logger.Errorf("%s stopped: %v", name, err)
		}

		select {
		case <-ctx.Done():
			return err
		default:
		}

		logger.Infof("%s will restart in %v", name, funcRestartWait)
		time.Sleep(funcRestartWait)
	}
}
