package tool

import "testing"

func TestGetChildPid(t *testing.T) {
	pid, err := GetChildPid(350)
	if err != nil {
		t.Fatal()
	}
	t.Log(pid)

	pid, err = GetChildPid(199)
	if err != nil {
		t.Fatal()
	}
	t.Log(pid)

	pid, err = GetChildPid(183)
	if err != nil {
		t.Fatal()
	}
	t.Log(pid)
}
