package cache

import (
	"context"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"time"
)

type informerCache struct {
	informerFactory informers.SharedInformerFactory
}

func (ic informerCache) Start(ctx context.Context) error {
	ic.informerFactory.Start(ctx.Done())
	return nil
}

func (ic informerCache) GetSharedInformerFactory() informers.SharedInformerFactory {
	return ic.informerFactory
}

func NewInformerCache(client kubernetes.Interface, defaultResync time.Duration, option informers.SharedInformerOption) Interface {
	ic := &informerCache{}
	ic.informerFactory = informers.NewSharedInformerFactoryWithOptions(client, defaultResync, option)
	return ic
}
