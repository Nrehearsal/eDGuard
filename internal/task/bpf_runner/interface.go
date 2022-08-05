package bpf_runner

import (
	"context"
	"eDGuard/internal/db"
)

type Interface interface {
	Start(ctx context.Context) error
	ShowAttachStatus()
	Add(t db.Interface)
	Delete(id string)
}
