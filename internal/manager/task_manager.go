package manager

import (
	"context"
	"eDGuard/internal/task/cache"
	"eDGuard/internal/task/pump"
	"eDGuard/pkg/cri"
	"eDGuard/pkg/k8s"
	"errors"
	"fmt"
	"github.com/cilium/ebpf"
	"github.com/cilium/ebpf/perf"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"
	"os"
	"sync"
	"time"
)

const NodeNameEnv = "MY_NODE_NAME"

type TaskManager struct {
	node string
	mu   sync.Mutex

	rd         *perf.Reader
	dbCtxQueue *ebpf.Map
	clusters   *ebpf.Map
	events     *ebpf.Map

	runnables *runnables

	criClient cri.Client
	k8sClient kubernetes.Interface
	cache     cache.Interface

	pump pump.Interface

	errCh          chan error
	internalCtx    context.Context
	internalCancel context.CancelFunc

	shutdownTimeout time.Duration
	shutdownCtx     context.Context
}

func NewTaskManager(events *ebpf.Map, clusters *ebpf.Map, dbCtx *ebpf.Map) Manager {
	m := TaskManager{
		events:          events,
		clusters:        clusters,
		dbCtxQueue:      dbCtx,
		errCh:           make(chan error),
		shutdownTimeout: time.Second * 5,
	}
	m.node = os.Getenv(NodeNameEnv)
	if m.node == "" {
		klog.Fatalf("evn %s must be specified\n", NodeNameEnv)
	}

	m.criClient = cri.GetClient()
	m.k8sClient = k8s.GetK8SClient()
	m.runnables = newRunnables(m.errCh)

	labelOptions := informers.WithTweakListOptions(func(options *metav1.ListOptions) {
		options.FieldSelector = fmt.Sprintf("spec.nodeName=%s", m.node)
	})
	m.cache = cache.NewInformerCache(m.k8sClient, time.Second*30, labelOptions)

	return &m
}

func (tm *TaskManager) Client() kubernetes.Interface {
	return tm.k8sClient
}

func (tm *TaskManager) Cri() cri.Client {
	return tm.criClient
}

func (tm *TaskManager) Cache() cache.Interface {
	return tm.cache
}

func (tm *TaskManager) AddTask(runnable Runnable) error {
	return tm.runnables.Add(runnable)
}

func (tm *TaskManager) Clusters() *ebpf.Map {
	return tm.clusters
}

func (tm *TaskManager) Start(ctx context.Context) error {
	tm.internalCtx, tm.internalCancel = context.WithCancel(ctx)

	// This chan indicates that stop is complete, in other words all runnables have returned or timeout on stop request
	stopComplete := make(chan struct{})
	defer close(stopComplete)
	// This must be deferred after closing stopComplete, otherwise we deadlock.
	defer func() {
		stopErr := tm.shutdown(stopComplete)
		if stopErr != nil {
			klog.Errorln("shutdown error:%s", stopErr)
		}
	}()

	// add informer
	err := tm.runnables.Add(tm.cache)
	if err != nil {
		klog.Errorf("failed to add cache runnable: %s", err)
		return err
	}

	// add pump
	tm.rd, err = perf.NewReader(tm.events, os.Getpagesize())
	if err != nil {
		klog.Errorf("create perf reader failed: %s", err)
		return err
	}
	tm.pump = pump.NewPump(tm.rd, tm.dbCtxQueue)
	err = tm.runnables.Add(tm.pump)
	if err != nil {
		klog.Errorf("failed to add pump runnable: %s", err)
		return err
	}

	// run pump
	if err := tm.runnables.Pumps.Start(tm.internalCtx); err != nil {
		return err
	}

	// run bpfprogram
	if err := tm.runnables.BpfPrograms.Start(tm.internalCtx); err != nil {
		return err
	}

	if err := tm.runnables.Caches.Start(tm.internalCtx); err != nil {
		return err
	}

	klog.Infof("ready")

	for {
		select {
		case <-ctx.Done():
			return nil
		case err := <-tm.errCh:
			return err
		}
	}
}

func (tm *TaskManager) shutdown(stopComplete <-chan struct{}) error {
	klog.Infof("shutdown")

	var shutdownCancel context.CancelFunc
	tm.shutdownCtx, shutdownCancel = context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	// Start draining the errors before acquiring the lock to make sure we don't deadlock
	// if something that has the lock is blocked on trying to write into the unbuffered
	// channel after something else already wrote into it.
	var closeOnce sync.Once
	go func() {
		for {
			// Closing in the for loop is required to avoid race conditions between
			// the closure of all internal procedures and making sure to have a reader off the error channel.
			closeOnce.Do(func() {
				// Cancel the internal stop channel and wait for the procedures to stop and complete.
				tm.internalCancel()
				tm.criClient.Close()
				_ = tm.rd.Close()
			})
			select {
			case err, ok := <-tm.errCh:
				if ok {
					klog.Error(err, "error received after stop sequence was engaged")
				}
			case <-stopComplete:
				return
			}
		}
	}()

	go func() {
		tm.runnables.Caches.StopAndWait(tm.shutdownCtx)
		tm.runnables.BpfPrograms.StopAndWait(tm.shutdownCtx)
		tm.runnables.Pumps.StopAndWait(tm.shutdownCtx)
		shutdownCancel()
	}()

	<-tm.shutdownCtx.Done()
	if err := tm.shutdownCtx.Err(); err != nil && err != context.Canceled {
		if errors.Is(err, context.DeadlineExceeded) {
			if tm.shutdownTimeout > 0 {
				return fmt.Errorf("failed waiting for all runnables to end within grace period of %s: %w", tm.shutdownTimeout, err)
			}
			return nil
		}
		// For any other error, return the error.
		return err
	}
	return nil
}
