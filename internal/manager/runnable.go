package manager

import (
	"context"
	"eDGuard/internal/db"
	"k8s.io/client-go/informers"
	"log"
)

// runnables handles all the runnables for a manager by grouping them accordingly to their
// type (webhooks, caches etc.).
type runnables struct {
	BpfPrograms *runnableGroup
	Caches      *runnableGroup
	Pumps       *runnableGroup
	HttpServers *runnableGroup
}

// newRunnables creates a new runnables object.
func newRunnables(errChan chan error) *runnables {
	return &runnables{
		BpfPrograms: newRunnableGroup(errChan),
		Caches:      newRunnableGroup(errChan),
		Pumps:       newRunnableGroup(errChan),
		HttpServers: newRunnableGroup(errChan),
	}
}

type bpfRunner interface {
	Runnable
	ShowAttachStatus()
	Add(t db.Interface)
	Delete(id string)
}

type hasCache interface {
	Runnable
	GetSharedInformerFactory() informers.SharedInformerFactory
}

type pumpRunner interface {
	Runnable
	PopOne()
}

// Add adds a runnable to closest group of runnable that they belong to.
//
// Add should be able to be called before and after Start, but not after StopAndWait.
// Add should return an error when called during StopAndWait.
// The runnables added before Start are started when Start is called.
// The runnables added after Start are started directly.
func (r *runnables) Add(fn Runnable) error {
	switch runnable := fn.(type) {
	case bpfRunner:
		return r.BpfPrograms.Add(fn, func(ctx context.Context) bool {
			return true
		})
	case hasCache:
		return r.Caches.Add(fn, func(ctx context.Context) bool {
			for informerType, ok := range runnable.GetSharedInformerFactory().WaitForCacheSync(ctx.Done()) {
				if !ok {
					log.Fatalf("failed to sync cache for %v", informerType)
					return false
				}
			}
			log.Printf("sync cache done")
			return true
		})
	case pumpRunner:
		return r.Caches.Add(fn, func(ctx context.Context) bool {
			return true
		})
	default:
		return nil
	}
}
