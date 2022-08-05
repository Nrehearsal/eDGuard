package cri

import (
	"context"
	dockerclient "github.com/docker/docker/client"
	"k8s.io/klog/v2"
	"os"
)

var dockerCRI *DockerCRI

type DockerCRI struct {
	client *dockerclient.Client
}

func (d *DockerCRI) GetContainerPid(id string) (int, error) {
	cinfo, err := d.client.ContainerInspect(context.TODO(), id)
	if err != nil {
		return 0, err
	}

	return cinfo.State.Pid, nil
}

func (d *DockerCRI) Close() {
	d.client.Close()
}

func GetDocker() Client {
	if dockerCRI == nil {
		klog.Fatal("uninitialized docker client")
	}
	return dockerCRI
}

func init() {
	if os.Getenv(CriName) == Docker {
		var err error
		dockerCRI = &DockerCRI{}

		dockerCRI.client, err = dockerclient.NewClientWithOpts(dockerclient.FromEnv, dockerclient.WithHost("unix:///var/run/cri.sock"), dockerclient.WithAPIVersionNegotiation())
		if err != nil {
			klog.Fatalf("unable to init a docker client")
		}
	}
}
