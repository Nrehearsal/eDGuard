package mysql

import (
	"eDGuard/internal/bpf"
	"fmt"
)

type Instance struct {
	clusterName string
	pod         string
	containerId string
	version     string
	pid         int
}

func NewMySQLTask(version string) *Instance {
	m := Instance{
		version: version,
	}
	return &m
}

func (mt *Instance) WithClusterName(clusterName string) *Instance {
	mt.clusterName = clusterName
	return mt
}

func (mt *Instance) WithPod(pod string) *Instance {
	mt.pod = pod
	return mt
}

func (mt *Instance) WithContainerId(containerId string) *Instance {
	mt.containerId = containerId
	return mt
}

func (mt *Instance) WithPid(pid int) *Instance {
	mt.pid = pid
	return mt
}

func (mt *Instance) GetContainerId() string {
	return mt.containerId
}

func (mt *Instance) GetPod() string {
	return mt.pod
}

func (mt *Instance) GetPid() int {
	return mt.pid
}

func (mt *Instance) GetClusterName() string {
	return mt.clusterName
}

func (mt *Instance) GetVersion() string {
	return mt.version
}

func (mt *Instance) GetKind() string {
	return bpf.MySQLKind
}

func (mt *Instance) GetId() string {
	return fmt.Sprintf("%s|%s", mt.version, mt.containerId)
}
