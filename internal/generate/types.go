package generate

import (
	"eDGuard/pkg/tool"
	"golang.org/x/sys/unix"
	"k8s.io/klog/v2"
	"time"
)

type BpfCluster struct {
	ClusterName [64]uint8
	Offsets     [32]uint64
}

type BpfDbCtx struct {
	Gtid              uint64
	CostTime          uint64
	ThdAddr           int64
	Timestamp         uint64
	Comm              [16]uint8
	ClusterName       [64]uint8
	User              [32]uint8
	PrivUser          [32]uint8
	Host              [64]uint8
	Ipaddr            [46]uint8
	Database          [46]uint8
	ThreadId          uint32
	QueryId           int64
	QueryLen          uint64
	Query             [1024]uint8
	PreviousFoundRows uint64
	CurrentFoundRows  uint64
	AffectedRows      uint64
	SentRowCount      uint64
	RowCountFunc      uint64
	SqlErrno          uint32
	Killed            int32
	PeerPort          uint16
	Processed         int8
	_                 [5]byte
}

type BpfEvent struct{ Gtid uint64 }

func (dbctx *BpfDbCtx) Print() {
	klog.Infof("date: %s\n", time.Unix(int64(dbctx.Timestamp), 0).Format(time.RFC3339))
	klog.Infof("cluster: %s\n", tool.B642S(dbctx.ClusterName))
	klog.Infof("process: %s\n", unix.ByteSliceToString(dbctx.Comm[:]))
	klog.Infof("process_id: %d\n", dbctx.Gtid>>32)
	klog.Infof("pthread_id: %d\n", dbctx.Gtid<<32>>32)
	klog.Infof("host: %s\n", unix.ByteSliceToString(dbctx.Host[:]))
	klog.Infof("ip: %s\n", unix.ByteSliceToString(dbctx.Ipaddr[:]))
	klog.Infof("client port: %d\n", dbctx.PeerPort)
	klog.Infof("thread_id: %d\n", dbctx.ThreadId)
	klog.Infof("database: %s\n", unix.ByteSliceToString(dbctx.Database[:]))
	klog.Infof("username: %s\n", unix.ByteSliceToString(dbctx.User[:]))
	klog.Infof("priv username: %s\n", unix.ByteSliceToString(dbctx.PrivUser[:]))
	klog.Infof("query_id: %d\n", dbctx.QueryId)
	klog.Infof("query: %s\n", unix.ByteSliceToString(dbctx.Query[:]))
	klog.Infof("query_len: %d\n", dbctx.QueryLen)
	klog.Infof("previous_found_rows: %d\n", dbctx.PreviousFoundRows)
	klog.Infof("current_found_rows: %d\n", dbctx.CurrentFoundRows)
	klog.Infof("affected_rows: %d\n", dbctx.AffectedRows)
	klog.Infof("sent_row_count: %d\n", dbctx.SentRowCount)
	klog.Infof("row_count_func: %d\n", dbctx.RowCountFunc)
	klog.Infof("sql_errno: %d\n", dbctx.SqlErrno)
	klog.Infof("killed: %d\n", dbctx.Killed)
	klog.Infof("cost_time: %d ns\n", dbctx.CostTime)
	klog.Infoln("##################################################################")
}
