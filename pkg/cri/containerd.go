package cri

import (
	"context"
	"github.com/containerd/containerd"
	"github.com/containerd/containerd/api/services/tasks/v1"
	"github.com/containerd/containerd/namespaces"
	"k8s.io/klog/v2"
	"os"
)

var containerdCRI *ContainerdCRI

type ContainerdCRI struct {
	client *containerd.Client
}

func (d *ContainerdCRI) GetContainerPid(id string) (int, error) {
	ctx := namespaces.WithNamespace(context.TODO(), "k8s.io")
	ctask, err := d.client.TaskService().Get(ctx, &tasks.GetRequest{ContainerID: id})
	if err != nil {
		klog.Errorf("get task for %s error:%s", id, err)
		return 0, err
	}
	return int(ctask.Process.Pid), nil
}

func (d *ContainerdCRI) Close() {
	d.client.Close()
}

func GetContainerd() Client {
	if containerdCRI == nil {
		klog.Fatal("uninitialized containerd client")
	}
	return containerdCRI
}

func init() {
	if os.Getenv(CriName) == Containerd {
		var err error
		containerdCRI = &ContainerdCRI{}

		containerdCRI.client, err = containerd.New("/var/run/cri.sock")
		if err != nil {
			klog.Fatalf("unable to init containerd client")
		}
	}
}
