package cache

import (
	"context"
	"k8s.io/client-go/informers"
)

type Interface interface {
	Start(ctx context.Context) error
	GetSharedInformerFactory() informers.SharedInformerFactory
}
