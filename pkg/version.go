package pkg

import (
	"fmt"
	"sort"
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

func FindHighestPatchVersion(tags []string, version string) (string, error) {
	var semverVersion Semver
	// Remove the last element if it's an empty string
	if tags[len(tags)-1] == "" {
		tags = tags[:len(tags)-1]
	}

	//sort tags in descending order
	sort.Strings(tags)
	sort.Slice(tags, func(i, j int) bool {
		return tags[i] > tags[j]
	})

	for _, tag := range tags {

		tagVersion, err := ParseSemver(tag)
		if err != nil {
			continue
		}

		if strings.Contains(version, ".") {
			requestedMajorMinor := fmt.Sprintf("%d.%d", tagVersion.Major, tagVersion.Minor)
			if requestedMajorMinor == version {
				semverVersion = tagVersion
				break
			}
		} else {
			if fmt.Sprintf("%d", tagVersion.Major) == version {
				semverVersion = tagVersion
				break
			}
		}
	}
	return fmt.Sprintf("v%d.%d.%d", semverVersion.Major, semverVersion.Minor, semverVersion.Patch), nil
}

func FormatVersion(version string) string {
	if strings.HasPrefix(version, "v") && !strings.Contains(version, ".") {
		version = version + ".0."
	}
	return version
}
