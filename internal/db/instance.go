package db

type Interface interface {
	GetClusterName() string
	GetPod() string
	GetContainerId() string
	GetPid() int
	GetVersion() string
	GetKind() string
	GetId() string
}
