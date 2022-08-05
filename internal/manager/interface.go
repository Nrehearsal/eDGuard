package manager

import (
	"context"
	"eDGuard/internal/task/cache"
	"eDGuard/pkg/cri"
	"github.com/cilium/ebpf"
	"k8s.io/client-go/kubernetes"
)

type Manager interface {
	Client() kubernetes.Interface
	Cri() cri.Client
	Cache() cache.Interface
	AddTask(runnable Runnable) error
	Start(ctx context.Context) error
	Clusters() *ebpf.Map
}
