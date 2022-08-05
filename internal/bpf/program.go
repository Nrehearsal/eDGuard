package bpf

import "github.com/cilium/ebpf"

type Program struct {
	Attached   bool
	Key        string
	Kind       string
	Version    string
	Upp        *ebpf.Program
	Urpp       *ebpf.Program
	Properties [32]uint64
}
