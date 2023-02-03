package svrctrl

import (
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/qumogu/go-tools/logger"
)

const (
	funcRestartWait = time.Second
)

// Trap 捕获信号.
func Trap(cleanup func()) {
	c := make(chan os.Signal, 1)

	signals := []os.Signal{os.Interrupt, syscall.SIGTERM}
	signal.Notify(c, signals...)

	const (
		MAXINTERRUPTCOOUNT = 3
		MININTERRUPTCOOUNT = 1
	)

	go func() {
		interruptCount := uint32(0) // 记录接收到信号的次数

		for sig := range c {
			go func(sig os.Signal) {
				logger.Infof("Received signal: '%v'", sig)

				switch sig {
				case os.Interrupt, syscall.SIGTERM:
					// 接收停止信号小于3次的，第一次清理工作然后退出进程，后面的信号不处理
					// 接收信号达到3次，直接退出，用于用户强制退出的场景
					if atomic.LoadUint32(&interruptCount) < MAXINTERRUPTCOOUNT {
						atomic.AddUint32(&interruptCount, MININTERRUPTCOOUNT)

						if atomic.LoadUint32(&interruptCount) == 1 {
							cleanup()
							os.Exit(0)
						} else {
							return
						}
					} else {
						logger.Info("Force stop, interrupting cleanup")
						os.Exit(128 + int(sig.(syscall.Signal)))
					}
				}
			}(sig)
		}
	}()
}
