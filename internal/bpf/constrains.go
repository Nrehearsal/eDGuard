package bpf

var MysqlProperties map[string][32]uint64

const MySQLKind = "MySQL"
const PerconaServer5734 = "percona-server:5.7.34"

func init() {
	/*
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
	*/
	// with 15 elements pads the last element is the offset for dispatch_command.
	MysqlProperties = map[string][32]uint64{}
	MysqlProperties[PerconaServer5734] = [32]uint64{
		8432, 8368, 544, 5120, 3936, 4096, 3968, 4000, 5112, 7600, 7608, 12776, 7640, 7624, 12768, 8564,
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0x867250}
}
