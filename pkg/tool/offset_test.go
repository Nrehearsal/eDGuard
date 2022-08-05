package tool

import "testing"

func TestGetOffsetBySymbol(t *testing.T) {
	off, err := GetOffsetBySymbol("/home/larry/mysqld", "_Z16dispatch_commandP3THDPK8COM_DATA19enum_server_command")
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("%x", off)
}
