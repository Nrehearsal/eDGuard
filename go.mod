module eDGuard

go 1.16

require (
	github.com/cilium/ebpf v0.9.0
	github.com/docker/docker v20.10.17+incompatible
	golang.org/x/sys v0.0.0-20220715151400-c0bba94af5f8
	sigs.k8s.io/controller-runtime v0.12.3
)

require (
	github.com/Microsoft/go-winio v0.5.2 // indirect
	github.com/containerd/containerd v1.6.6
	github.com/docker/distribution v2.8.1+incompatible // indirect
	k8s.io/api v0.24.2
	k8s.io/apimachinery v0.24.2
	k8s.io/client-go v0.24.2
	k8s.io/klog/v2 v2.60.1
)
