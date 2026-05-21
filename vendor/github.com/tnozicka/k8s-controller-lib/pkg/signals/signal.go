package signals

import (
	"context"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/klog/v2"
)

var (
	stopChannel = make(chan struct{})
	once        sync.Once

	shutdownSignals = []os.Signal{syscall.SIGINT, syscall.SIGABRT, syscall.SIGTERM}
)

func setupStopChannel() {
	c := make(chan os.Signal, 2)
	signal.Notify(c, shutdownSignals...)
	go func() {
		s := <-c
		klog.InfoS("Received shutdown signal, shutting down...", "Signal", s)
		close(stopChannel)
		<-c
		klog.InfoS("Received second shutdown signal, exiting...", "Signal", s)
		os.Exit(1)
	}()
}

func GetStopChannel() (stopCh <-chan struct{}) {
	once.Do(setupStopChannel)
	return stopChannel
}

func GetContext() context.Context {
	return wait.ContextForChannel(GetStopChannel())
}
