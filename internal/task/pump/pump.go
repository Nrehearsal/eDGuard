package pump

import (
	"context"
	"eDGuard/internal/generate"
	"errors"
	"github.com/cilium/ebpf"
	"github.com/cilium/ebpf/perf"
	"k8s.io/klog/v2"
)

type eventPump struct {
	rd    *perf.Reader
	dbCtx *ebpf.Map
}

func NewPump(rd *perf.Reader, dbCtx *ebpf.Map) Interface {
	return &eventPump{
		rd:    rd,
		dbCtx: dbCtx,
	}
}

func (p *eventPump) PopOne() {
}

func (p *eventPump) Start(ctx context.Context) error {
	klog.Infof("Listening for events..")

	var err error
	var record perf.Record
	var dbCtx generate.BpfDbCtx

	//p.rd, err = perf.NewReader(p.events, os.Getpagesize())
	//if err != nil {
	//	klog.Errorf("creating perf event reader: %s", err)
	//	return err
	//}

	//defer func(rd *perf.Reader) {
	//	err = rd.Close()
	//	if err != nil {
	//		klog.Errorf("warnning perf.Reader close failed")
	//	}
	//}(m.rd)

	for {
		select {
		case <-ctx.Done():
			klog.Infof("pump loop quit due to context done.")
			return nil
		default:
			record, err = p.rd.Read()
			if err != nil {
				klog.Errorf("reading from perf event reader: %s", err)
				if errors.Is(err, perf.ErrClosed) {
					return err
				}
				continue
			}

			if record.LostSamples != 0 {
				klog.Warningf("perf event ring buffer full, dropped %d samples", record.LostSamples)
				continue
			}

			//// Parse the perf event entry into a bpfEvent structure.
			//if err := binary.Read(bytes.NewBuffer(record.RawSample), binary.LittleEndian, &event); err != nil {
			//	klog.Printf("parsing perf event: %s", err)
			//}

			if err = p.dbCtx.LookupAndDelete(nil, &dbCtx); err != nil {
				if !errors.Is(err, ebpf.ErrKeyNotExist) {
					return err
				}
			}
			dbCtx.Print()
		}
	}
}
