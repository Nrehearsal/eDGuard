package tool

import (
	"eDGuard/pkg/cri"
	"strings"
)

func ParseImage(image string) string {
	parts := strings.Split(image, "/")
	length := len(parts)

	if length == 1 {
		return parts[0]
	}

	return parts[length-1]
}

func ParseContainerId(containerId string) string {
	//var schema, id string
	parts := strings.Split(containerId, "://")
	length := len(parts)

	if length == 1 {
		return parts[0]
	}

	if length == 2 {
		if parts[0] == cri.Docker {
			return parts[1][:12]
		}
		return parts[1]
	}

	return containerId
}
