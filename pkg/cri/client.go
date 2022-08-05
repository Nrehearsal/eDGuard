package cri

import "os"

const Docker = "docker"
const Containerd = "containerd"
const CriName = "CRI_NAME"

type Client interface {
	GetContainerPid(id string) (int, error)
	Close()
}

func GetClient() Client {
	if os.Getenv(CriName) == Docker {
		return GetDocker()
	}
	return GetContainerd()
}
