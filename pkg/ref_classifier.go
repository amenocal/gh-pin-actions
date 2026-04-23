package pkg

import (
	"regexp"
	"strings"
)

type UsesRefClass string

const (
	UsesRefLocal       UsesRefClass = "local"
	UsesRefVersion     UsesRefClass = "version"
	UsesRefBranch      UsesRefClass = "branch"
	UsesRefSHA         UsesRefClass = "sha"
	UsesRefUnsupported UsesRefClass = "unsupported"
)

type UsesRefMetadata struct {
	Class      UsesRefClass
	ActionPath string
	BaseRepo   string
	Ref        string
}

var (
	versionRefPattern = regexp.MustCompile(`^v?\d+(?:\.\d+){0,2}$`)
	shaRefPattern     = regexp.MustCompile(`^[a-fA-F0-9]{40}$`)
)

func ClassifyUsesRef(uses string) UsesRefMetadata {
	uses = strings.TrimSpace(uses)
	if uses == "" {
		return UsesRefMetadata{Class: UsesRefUnsupported}
	}

	if strings.HasPrefix(uses, "docker://") {
		return UsesRefMetadata{
			Class:      UsesRefUnsupported,
			ActionPath: uses,
		}
	}

	actionPath, ref, _ := splitOnLastAt(uses)

	if strings.HasPrefix(actionPath, "./") {
		return UsesRefMetadata{
			Class:      UsesRefLocal,
			ActionPath: actionPath,
			Ref:        ref,
		}
	}

	baseRepo, _ := ExtractOwnerRepo(actionPath)
	result := UsesRefMetadata{
		ActionPath: actionPath,
		BaseRepo:   baseRepo,
		Ref:        ref,
	}

	if baseRepo == "" || actionPath == "" || ref == "" {
		result.Class = UsesRefUnsupported
		return result
	}

	switch {
	case shaRefPattern.MatchString(ref):
		result.Class = UsesRefSHA
	case versionRefPattern.MatchString(ref):
		result.Class = UsesRefVersion
	default:
		result.Class = UsesRefBranch
	}

	return result
}
