package tool

import "testing"

func TestParseContainerId(t *testing.T) {
	id := ParseContainerId("docker://bbccb2a9ecfcff52751855a75d6f317dcb0d685c160b15cf0919f1c9d276aaae")
	t.Log(id)

	id = ParseContainerId("containerd://bbccb2a9ecfcff52751855a75d6f317dcb0d685c160b15cf0919f1c9d276aaae")
	t.Log(id)
}
