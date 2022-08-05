package mysql

import (
	"context"
	"eDGuard/internal/bpf"
	"eDGuard/internal/db"
	"eDGuard/internal/db/mysql"
	"eDGuard/internal/generate"
	"eDGuard/internal/manager"
	"eDGuard/internal/task/bpf_runner"
	"eDGuard/pkg/cri"
	"eDGuard/pkg/tool"
	"fmt"
	"github.com/cilium/ebpf"
	"github.com/cilium/ebpf/link"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"
	"strings"
	"sync"
)

type BpfRunner struct {
	Kind string

	mu *sync.RWMutex
	wg *sync.WaitGroup

	Instances   map[string]db.Interface
	bpfPrograms map[string]*bpf.Program

	clusters *ebpf.Map

	internalErrChan chan error
	// ch is the internal channel where the pids are read off from.
	ch chan db.Interface

	informerFactory informers.SharedInformerFactory
	criClient       cri.Client

	internalContext context.Context
	internalCancel  context.CancelFunc
}

func NewFromManager(mgr manager.Manager) *BpfRunner {
	r := &BpfRunner{
		Kind: bpf.MySQLKind,

		mu: new(sync.RWMutex),
		wg: new(sync.WaitGroup),

		Instances:   map[string]db.Interface{},
		bpfPrograms: make(map[string]*bpf.Program),

		clusters: mgr.Clusters(),

		internalErrChan: make(chan error),
		ch:              make(chan db.Interface),
		criClient:       mgr.Cri(),
		informerFactory: mgr.Cache().GetSharedInformerFactory(),
	}

	r.informerFactory.Core().V1().Pods().Informer().AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc:    r.onPodCreate,
			UpdateFunc: r.onPodUpdate,
			DeleteFunc: r.onPodDelete,
		})
	return r
}

func (r *BpfRunner) WithBpfProgram(p *bpf.Program) *BpfRunner {
	r.bpfPrograms[p.Version] = p
	return r
}

func (r *BpfRunner) Complete() bpf_runner.Interface {
	return r
}

func (r *BpfRunner) Start(ctx context.Context) error {
	klog.Infof("[%s] start", r.Kind)

	r.internalContext, r.internalCancel = context.WithCancel(ctx)

	defer func() {
		klog.Infof("[%s] exited", r.Kind)
		r.internalCancel()
		r.wg.Wait()
		close(r.ch)
		close(r.internalErrChan)
	}()

	for {
		select {
		case <-ctx.Done():
			return nil
		case t := <-r.ch:
			r.attachInstance(t)
		case err := <-r.internalErrChan:
			klog.Errorf("[%s] has error: %s", err)
		}
	}
}

func (r *BpfRunner) ShowAttachStatus() {
}

func (r *BpfRunner) Add(t db.Interface) {
	r.ch <- t
}

func (r *BpfRunner) Delete(id string) {
	r.detachInstance(id)
}

func (r *BpfRunner) attachInstance(t db.Interface) {
	r.mu.Lock()
	defer r.mu.Unlock()

	bpfProgram := r.bpfPrograms[t.GetVersion()]

	cluster := generate.BpfCluster{
		ClusterName: tool.S264B(t.GetClusterName()),
		Offsets:     bpfProgram.Properties,
	}

	r.Instances[t.GetId()] = t

	if err := r.clusters.Update(uint32(t.GetPid()), &cluster, ebpf.UpdateAny); err != nil {
		r.internalErrChan <- err
		return
	}

	if !bpfProgram.Attached {
		r.wg.Add(1)
		go r.attach(r.internalContext, t.GetPid(), bpfProgram)
	}
}

func (r *BpfRunner) detachInstance(id string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if t, exists := r.Instances[id]; exists {
		if err := r.clusters.Delete(uint32(t.GetPid())); err != nil {
			r.internalErrChan <- err
			return
		}
		delete(r.Instances, id)
	}
}

func (r *BpfRunner) attach(ctx context.Context, pid int, program *bpf.Program) {
	defer r.wg.Done()

	klog.Infof("[%s][%s] attach ebpf program", r.Kind, program.Version)

	binPath := fmt.Sprintf("/proc/%d/exe", pid)

	// Open an ELF binary and read its symbols.
	ex, err := link.OpenExecutable(binPath)
	if err != nil {
		r.internalErrChan <- err
		return
	}

	// Open a Uretprobe at the exit point of the symbol and attach
	// the pre-compiled eBPF program to it.
	up, err := ex.Uprobe("_[BY_OFFSET]_", program.Upp, &link.UprobeOptions{Offset: program.Properties[31]})
	if err != nil {
		r.internalErrChan <- err
		return
	}
	defer up.Close()

	urp, err := ex.Uretprobe("_[BY_OFFSET]_", program.Urpp, &link.UprobeOptions{Offset: program.Properties[31]})
	if err != nil {
		r.internalErrChan <- err
		return
	}
	defer urp.Close()

	program.Attached = true

	select {
	case <-ctx.Done():
		klog.Infof("[%s][%s] un-attach ebpf program due to context done", r.Kind, program.Version)
		return
	}
}

func (r *BpfRunner) onPodCreate(obj interface{}) {
	r.onPodUpdate(nil, obj)
}

func (r *BpfRunner) onPodUpdate(old, new interface{}) {
	pod := new.(*corev1.Pod)
	//klog.Infof("[%s] new pod update event: %s", r.Kind, pod.Name)

	if pod.Status.Phase != corev1.PodRunning {
		return
	}

	if v, ok := pod.Labels["app.kubernetes.io/managed-by"]; !ok || v != "mysql.radondb.com" {
		return
	}

	for _, c := range pod.Status.ContainerStatuses {
		if !c.Ready {
			continue
		}
		if !(strings.Contains(c.Image, "percona") || strings.Contains(c.Image, "mysql")) {
			continue
		}

		imageId := tool.ParseImage(c.Image)
		containerId := tool.ParseContainerId(c.ContainerID)
		id := fmt.Sprintf("%s|%s", imageId, containerId)

		if _, ok := r.bpfPrograms[imageId]; !ok {
			continue
		}

		_, exists := r.Instances[id]
		if exists {
			continue
		}

		pid, err := r.criClient.GetContainerPid(containerId)
		if err != nil {
			r.internalErrChan <- err
			break
		}
		pid, err = tool.GetChildPid(pid)
		if err != nil {
			r.internalErrChan <- err
			break
		}

		klog.Infof("[%s][%s][%s] prepared to attach bpf program", r.Kind, pod.Name, id)
		mt := mysql.NewMySQLTask(imageId).
			WithClusterName(pod.Labels["app.kubernetes.io/instance"]).WithPod(pod.Name).
			WithContainerId(containerId).WithPid(pid)
		r.Add(mt)
		break
	}
}

func (r *BpfRunner) onPodDelete(obj interface{}) {
	pod := obj.(*corev1.Pod)

	for _, c := range pod.Status.ContainerStatuses {
		if !(strings.Contains(c.Image, "percona") || strings.Contains(c.Image, "mysql")) {
			continue
		}

		imageId := tool.ParseImage(c.Image)
		containerId := tool.ParseContainerId(c.ContainerID)
		id := fmt.Sprintf("%s|%s", imageId, containerId)

		_, ok := r.bpfPrograms[imageId]
		if !ok {
			continue
		}

		_, exists := r.Instances[id]
		if !exists {
			continue
		}

		klog.Infof("[%s][%s][%s] will be un-attach bpf program", r.Kind, pod.Name, id)
		r.Delete(id)

		break
	}
}
