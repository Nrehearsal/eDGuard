// +build ignore

#include "common.h"
#include "bpf_tracing.h"

#include "mysql.h"

char __license[] SEC("license") = "Dual MIT/GPL";

enum field_offset {
    m_thread_id_offset,
    m_query_id_offset,
    m_db_offset,
    tv_sec_offset,
    m_user_offset,
    m_priv_user_offset,
    m_host_offset,
    m_ip_offset,
    peer_port_offset,
    previous_found_rows_offset,
    current_found_rows_offset,
    m_affected_rows_offset,
    m_sent_row_count_offset,
    m_row_count_func_offset,
    m_mysql_errno_offset,
    killed_offset,
};

struct event {
    u64 gtid;
};

#define MAX_COMM_LEN 16
#define MAX_CLUSTER_NAME_LEN 64
#define MAX_USER_LEN 32
#define MAX_HOST_LEN 64
#define MAX_IP_LEN 46
#define MAX_DATABASE_LEN 64
#define MAX_QUERY_LEN 1024

struct db_ctx {
    u64 gtid;
    u64 cost_time;
    s64 thd_addr;
    ulong timestamp;
    u8 comm[MAX_COMM_LEN];
    u8 cluster_name[MAX_CLUSTER_NAME_LEN];
    u8 user[MAX_USER_LEN];
    u8 priv_user[MAX_USER_LEN];
    u8 host[MAX_HOST_LEN];
    u8 ipaddr[MAX_IP_LEN];
    u8 database[MAX_IP_LEN];
    u32 thread_id;
    s64 query_id;
    u64 query_len;
    u8 query[MAX_QUERY_LEN];
    u64 previous_found_rows;
    u64 current_found_rows;
    u64 affected_rows;
    ulong sent_row_count;
    u64 row_count_func;
    u32 sql_errno;
    s32 killed;
    u16 peer_port;
    s8 processed;
};

struct cluster {
    u8 cluster_name[64];
    u64 offsets[32];
};

struct {
	__uint(type, BPF_MAP_TYPE_PERCPU_ARRAY);
	__type(key, u32);
	__type(value, struct db_ctx);
	__uint(max_entries, 1);
} var_holder SEC(".maps");

struct {
    __uint(type, BPF_MAP_TYPE_HASH);
    __type(key, u32);
    __type(value, struct cluster);
    __uint(max_entries, 1024);
} clusters SEC(".maps");

struct {
    __uint(type, BPF_MAP_TYPE_HASH);
    __type(key, u32);
    __type(value, struct db_ctx);
    __uint(max_entries, 1024);
} thread_db_ctx_hash SEC(".maps");

struct {
    __uint(type, BPF_MAP_TYPE_QUEUE);
    __type(value, struct db_ctx);
    __uint(max_entries, 5120);
//    __uint(pinning, LIBBPF_PIN_BY_NAME);
} db_ctx_queue SEC(".maps");

struct {
    __uint(type, BPF_MAP_TYPE_PERF_EVENT_ARRAY);
} events SEC(".maps");

const struct event *unused __attribute__((unused));

SEC("uprobe/dispatch_command_57")
int mysql57_query(struct pt_regs *ctx) {
    u64 gtid = bpf_get_current_pid_tgid();

    struct cluster *cluster = bpf_map_lookup_elem(&clusters, &(u32){gtid>>32});
    if (!cluster) {
    #ifdef DEBUG_ENABLE
        bpf_printk("U[%d][%d] not found in clusters map, skipped\n", gtid>>32, gtid<<32>>32);
    #endif
        return 0;
    }

    u64 command = (u64)PT_REGS_PARM3(ctx);
    if (command != COM_QUERY) {
        return 0;
    }

    void *arg = NULL;
    void *cnt = NULL;

    struct db_ctx *db_ctx = bpf_map_lookup_elem(&var_holder, &(u32){0});
    if (!db_ctx) {
    #ifdef DEBUG_ENABLE
        bpf_printk("U[error]db_ctx not found in var_holder map\n");
    #endif
        return 0;
    }

    // set gtid
    db_ctx->gtid = gtid;

    // record current kernel time in ns
    db_ctx->cost_time = bpf_ktime_get_ns();

    // get process name
    bpf_get_current_comm(&db_ctx->comm, sizeof(db_ctx->comm));

    // record cluster name
    bpf_probe_read_user(&db_ctx->cluster_name, sizeof(db_ctx->cluster_name), &cluster->cluster_name);

    // get COM_DATA *com_data
    arg = (void *)PT_REGS_PARM2(ctx);
    // parse COM_DATA
    struct COM_QUERY_DATA query;
    bpf_probe_read_user(&query, sizeof(query), arg);
    bpf_probe_read_user(&db_ctx->query, sizeof(db_ctx->query), query.query);
    bpf_probe_read_user(&db_ctx->query_len, sizeof(db_ctx->query_len), &query.length);

    // get THD *thd
    arg = (void *)PT_REGS_PARM1(ctx);
    // read thread_id, m_thread_id
    db_ctx->thd_addr = (u64)arg;
    bpf_probe_read_user(&db_ctx->thread_id, sizeof(db_ctx->thread_id), arg+cluster->offsets[m_thread_id_offset]);
    // read query_id, query_id
    bpf_probe_read_user(&db_ctx->query_id, sizeof(db_ctx->query_id), arg+cluster->offsets[m_query_id_offset]);

    // read database, m_db
    struct st_mysql_const_lex_string  m_db;
    bpf_probe_read_user(&m_db, sizeof(m_db), arg+cluster->offsets[m_db_offset]);
    bpf_probe_read_user(&db_ctx->database, sizeof(db_ctx->database), m_db.str);

    // read start time, tv_sec
    bpf_probe_read_user(&db_ctx->timestamp, sizeof(db_ctx->timestamp), arg+cluster->offsets[tv_sec_offset]);

    // read username, m_user
    bpf_probe_read_user(&cnt, 8, arg+cluster->offsets[m_user_offset]);
    bpf_probe_read_str(&db_ctx->user, sizeof(db_ctx->user), cnt);

    // read priv username, m_priv_user
    bpf_probe_read_user(&cnt, 8, arg+cluster->offsets[m_priv_user_offset]);
    bpf_probe_read_str(&db_ctx->priv_user, sizeof(db_ctx->priv_user), cnt);

    // read host, m_host
    bpf_probe_read_user(&cnt, 8, arg+cluster->offsets[m_host_offset]);
    bpf_probe_read_user(&db_ctx->host, sizeof(db_ctx->host), cnt);

    // read ip, m_ip
    bpf_probe_read_user(&cnt, 8, arg+cluster->offsets[m_ip_offset]);
    bpf_probe_read_user(&db_ctx->ipaddr, sizeof(db_ctx->ipaddr), cnt);

    // read peer port, peer_port
    bpf_probe_read_user(&db_ctx->peer_port, sizeof(db_ctx->peer_port), arg+cluster->offsets[peer_port_offset]);

    // mark unhandle uretprobe
    db_ctx->processed = 0xf;

    #ifdef DEBUG_ENABLE
    #endif

    bpf_map_update_elem(&thread_db_ctx_hash, &db_ctx->gtid, db_ctx, BPF_ANY);

    return 0;
}

SEC("uretprobe/dispatch_command_57")
int mysql57_query_return(struct pt_regs *ctx) {
    struct event et = {};
    et.gtid = bpf_get_current_pid_tgid();

    struct cluster *cluster = bpf_map_lookup_elem(&clusters, &(u32){et.gtid>>32});
    if (!cluster) {
    #ifdef DEBUG_ENABLE
        bpf_printk("UR[%d][%d] not found in clusters map, skipped\n", et.gtid>>32, et.gtid<<32>>32);
    #endif
        return 0;
    }

    struct db_ctx *db_ctx = bpf_map_lookup_elem(&thread_db_ctx_hash, &et.gtid);
    if (!db_ctx) {
    #ifdef DEBUG_ENABLE
        bpf_printk("UR[%d][%d] not found in thread_db_ctx_hash map, skipped.\n", et.gtid>>32, et.gtid<<32>>32);
    #endif
        return 0;
    }
    if (db_ctx->processed != 0xf) {
     #ifdef DEBUG_ENABLE
        bpf_printk("UR[%d][%d] has already been processed, skipped.\n", et.gtid>>32, et.gtid<<32>>32);
     #endif
        return 0;
    }

    void *arg = (void *)db_ctx->thd_addr;

    // read previous_found_rows, previous_found_rows
    bpf_probe_read_user(&db_ctx->previous_found_rows, sizeof(db_ctx->previous_found_rows), arg+cluster->offsets[previous_found_rows_offset]);
    // read current_found_rows, current_found_rows
    bpf_probe_read_user(&db_ctx->current_found_rows, sizeof(db_ctx->current_found_rows), arg+cluster->offsets[current_found_rows_offset]);
    // read affected_rows, m_affected_rows
    bpf_probe_read_user(&db_ctx->affected_rows, sizeof(db_ctx->affected_rows), arg+cluster->offsets[m_affected_rows_offset]);
    // read sent_row_count, m_sent_row_count
    bpf_probe_read_user(&db_ctx->sent_row_count, sizeof(db_ctx->sent_row_count), arg+cluster->offsets[m_sent_row_count_offset]);
    // read row_count_func, m_row_count_func
    bpf_probe_read_user(&db_ctx->row_count_func, sizeof(db_ctx->row_count_func), arg+cluster->offsets[m_row_count_func_offset]);
    // read mysql_errno, m_mysql_errno
    bpf_probe_read_user(&db_ctx->sql_errno, sizeof(db_ctx->sql_errno), arg+cluster->offsets[m_mysql_errno_offset]);
    // read killed, killed
    bpf_probe_read_user(&db_ctx->killed, sizeof(db_ctx->killed), arg+cluster->offsets[killed_offset]);

    // calc cost time
    db_ctx->cost_time = bpf_ktime_get_ns() - db_ctx->cost_time;

    // mark processing complete
    db_ctx->processed = 0;

    bpf_map_push_elem(&db_ctx_queue, db_ctx, BPF_EXIST);

    bpf_perf_event_output(ctx, &events, BPF_F_CURRENT_CPU, &et, sizeof(et));

    return 0;
}