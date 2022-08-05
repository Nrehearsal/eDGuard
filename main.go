//go:build linux
// +build linux

// This program demonstrates how to attach an eBPF program to a uretprobe.
// The program will be attached to the 'readline' symbol in the binary '/bin/bash' and print out
// the line which 'readline' functions returns to the caller.
package main

import (
	"eDGuard/internal/bpf"
	"eDGuard/internal/manager"
	"eDGuard/internal/task/bpf_runner/mysql"
	"github.com/cilium/ebpf"
	"github.com/cilium/ebpf/rlimit"
	klog "k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
)

// $BPF_CLANG and $BPF_CFLAGS are set by the Makefile.
////go:generate go run github.com/cilium/ebpf/cmd/bpf2go -cc $BPF_CLANG -cflags $BPF_CFLAGS -target native -type event bpf uretprobe.c -- -I./headers
//go:generate go run github.com/cilium/ebpf/cmd/bpf2go -cc $BPF_CLANG -cflags $BPF_CFLAGS -target native -type event bpf mysqld_probe.c -- -I./headers

func main() {
	objs := bpfInit()
	defer func() {
		klog.Info("close bpf object")
		err := objs.Close()
		if err != nil {
			klog.Errorf("failed to close bpf object: %s", objs)
		}
	}()

	mgr := manager.NewTaskManager(objs.Events, objs.Clusters, objs.DbCtxQueue)

	percona5734 := &bpf.Program{Key: bpf.PerconaServer5734, Kind: bpf.MySQLKind, Version: bpf.PerconaServer5734,
		Upp: objs.Mysql57Query, Urpp: objs.Mysql57QueryReturn, Properties: bpf.MysqlProperties[bpf.PerconaServer5734]}

	mysqlTask := mysql.NewFromManager(mgr).
		WithBpfProgram(percona5734).
		Complete()

	if err := mgr.AddTask(mysqlTask); err != nil {
		klog.Fatalf("unable to add task[%v] to task manager", mysqlTask)
	}

	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		klog.Errorf("task manager exit with error: %s", err)
	}
}

func bpfInit() bpfObjects {
	// TODO remove it
	//pinPath := path.Join(bpfFSPath, "mysql57")
	//if err := os.MkdirAll(pinPath, os.ModePerm); err != nil {
	//	log.Fatalf("failed to create bpf fs subpath: %+v", err)
	//}

	// Load pre-compiled programs and maps into the kernel.
	//objs := bpfObjects{}
	//if err := loadBpfObjects(&objs, &ebpf.CollectionOptions{Maps: ebpf.MapOptions{PinPath: pinPath}}); err != nil {
	//	log.Fatalf("loading objects: %s", err)
	//}
	//defer objs.Close()

	// Allow the current process to lock memory for eBPF resources.
	if err := rlimit.RemoveMemlock(); err != nil {
		klog.Fatal("unable to remove mem lock", err)
	}

	// Load pre-compiled programs and maps into the kernel.
	objs := bpfObjects{}
	if err := loadBpfObjects(&objs, &ebpf.CollectionOptions{}); err != nil {
		klog.Fatalf("loading objects: %s", err)
	}

	pads := make([]bpfDbCtx, 1)
	padKey := uint32(0)
	err := objs.VarHolder.Update(&padKey, pads, ebpf.UpdateAny)
	if err != nil {
		klog.Fatalf("unable to init per cpu map: %s", err)
	}

	klog.Infof("map: %s, pined: %t", objs.DbCtxQueue.String(), objs.DbCtxQueue.IsPinned())

	return objs
}
