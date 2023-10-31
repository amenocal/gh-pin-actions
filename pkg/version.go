package pkg

import (
	"fmt"
	"strconv"
	"strings"
)

type Semver struct {
	Major int
	Minor int
	Patch int
}

func ParseSemver(version string) (Semver, error) {
	version = strings.TrimPrefix(version, "v")
	parts := strings.Split(version, ".")
	if len(parts) != 3 {
		return Semver{}, fmt.Errorf("invalid semver: %s", version)
	}

	major, err := strconv.Atoi(parts[0])
	if err != nil {
		return Semver{}, err
	}

	minor, err := strconv.Atoi(parts[1])
	if err != nil {
		return Semver{}, err
	}

	patch, err := strconv.Atoi(parts[2])
	if err != nil {
		return Semver{}, err
	}

	return Semver{Major: major, Minor: minor, Patch: patch}, nil
}

func ProcessActionsVersion(version string) string {
	if strings.HasPrefix(version, "v") && !strings.Contains(version, ".") {
		version = version + ".0."
	}
	return version
}
