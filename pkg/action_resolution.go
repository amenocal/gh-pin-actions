package pkg

import (
	"fmt"
	"strings"
)

// ActionVersionResolver resolves a repository + version to (commitSHA, tagVersion, error).
// Maps directly to GetActionHashByVersion in cmd/root.go.
type ActionVersionResolver func(repository, version string) (string, string, error)

// ActionBranchResolver resolves a repository + branch to (commitSHA, error).
// Maps directly to GetBranchHash in cmd/root.go.
type ActionBranchResolver func(repository, branch string) (string, error)

// ResolutionFailureWarning is called when resolution fails, allowing the caller
// to log a warning without halting processing of remaining actions.
type ResolutionFailureWarning func(action string, err error)

type actionLookupKind string

const (
	actionLookupVersion actionLookupKind = "version"
	actionLookupBranch  actionLookupKind = "branch"
)

// ActionResolutionCacheKey identifies a unique resolver lookup so duplicate
// API calls across workflow files/jobs can be avoided.
type ActionResolutionCacheKey struct {
	Repository string
	Mode       string
	Kind       actionLookupKind
	Ref        string
}

type actionResolutionCacheEntry struct {
	commitSHA   string
	resolvedRef string
	err         error
}

// ActionResolutionCache stores resolver results keyed by (repo, mode, kind, ref).
type ActionResolutionCache map[ActionResolutionCacheKey]actionResolutionCacheEntry

func NewActionResolutionCache() ActionResolutionCache {
	return make(ActionResolutionCache)
}

func formatPinnedAction(actionPath, commitSHA, resolvedRef string) string {
	return fmt.Sprintf("%s@%s #%s", actionPath, commitSHA, resolvedRef)
}

func lookupModeName(latestMode bool) string {
	if latestMode {
		return "latest"
	}
	return "default"
}

// normalizedVersionLookupRef determines which version string to pass to
// GetActionHashByVersion: "latest" in latest mode, or the formatted version.
func normalizedVersionLookupRef(ref string, latestMode bool) (string, bool) {
	if latestMode {
		return "latest", true
	}
	versionToResolve := strings.TrimPrefix(strings.TrimSpace(ref), "v")
	if versionToResolve == "" {
		return "", false
	}
	return FormatVersion(versionToResolve), true
}

func buildResolutionCacheKey(action UsesRefMetadata, latestMode bool) (ActionResolutionCacheKey, bool) {
	if action.BaseRepo == "" || action.ActionPath == "" || action.Ref == "" {
		return ActionResolutionCacheKey{}, false
	}

	switch action.Class {
	case UsesRefVersion:
		versionRef, ok := normalizedVersionLookupRef(action.Ref, latestMode)
		if !ok {
			return ActionResolutionCacheKey{}, false
		}
		return ActionResolutionCacheKey{
			Repository: action.BaseRepo,
			Mode:       lookupModeName(latestMode),
			Kind:       actionLookupVersion,
			Ref:        versionRef,
		}, true
	case UsesRefSHA:
		if !latestMode {
			return ActionResolutionCacheKey{}, false
		}
		return ActionResolutionCacheKey{
			Repository: action.BaseRepo,
			Mode:       lookupModeName(latestMode),
			Kind:       actionLookupVersion,
			Ref:        "latest",
		}, true
	case UsesRefBranch:
		return ActionResolutionCacheKey{
			Repository: action.BaseRepo,
			Mode:       lookupModeName(latestMode),
			Kind:       actionLookupBranch,
			Ref:        strings.TrimSpace(action.Ref),
		}, true
	default:
		return ActionResolutionCacheKey{}, false
	}
}

// resolveAndPin calls GetActionHashByVersion or GetBranchHash directly
// based on the ref type — no intermediate wrapper layers.
func resolveAndPin(refMeta UsesRefMetadata, latestMode bool, resolveByVersion ActionVersionResolver, resolveByBranch ActionBranchResolver) (commitSHA, resolvedRef string, err error) {
	switch refMeta.Class {
	case UsesRefVersion, UsesRefSHA:
		// For SHA refs, always resolve as "latest"; for version refs, respect latestMode
		effectiveLatest := latestMode || refMeta.Class == UsesRefSHA
		versionRef, ok := normalizedVersionLookupRef(refMeta.Ref, effectiveLatest)
		if !ok {
			return "", "", fmt.Errorf("invalid action format: %s@%s", refMeta.ActionPath, refMeta.Ref)
		}
		commitSHA, resolvedRef, err = resolveByVersion(refMeta.BaseRepo, versionRef)
		if err != nil {
			return "", "", err
		}
		return strings.TrimSpace(commitSHA), strings.TrimSpace(resolvedRef), nil

	case UsesRefBranch:
		branch := strings.TrimSpace(refMeta.Ref)
		commitSHA, err = resolveByBranch(refMeta.BaseRepo, branch)
		if err != nil {
			return "", "", err
		}
		return strings.TrimSpace(commitSHA), branch, nil

	default:
		return "", "", fmt.Errorf("unsupported action format: %s@%s", refMeta.ActionPath, refMeta.Ref)
	}
}

// ProcessActionWithCache is the single entry point for resolving and pinning an action.
// It classifies the ref, checks the cache, calls the resolver (GetActionHashByVersion
// or GetBranchHash), caches the result, and returns the pinned string.
// On failure it warns and returns ("", false, nil) so processing continues.
func ProcessActionWithCache(action string, latestMode bool, resolutionCache ActionResolutionCache, resolveByVersion ActionVersionResolver, resolveByBranch ActionBranchResolver, warnResolutionFailure ResolutionFailureWarning) (string, bool, error) {
	refMeta := ClassifyUsesRef(action)

	// Skip refs we don't resolve
	switch refMeta.Class {
	case UsesRefLocal, UsesRefUnsupported:
		return "", false, nil
	case UsesRefSHA:
		if !latestMode {
			return "", false, nil
		}
	}

	// Check cache
	cacheKey, cacheable := buildResolutionCacheKey(refMeta, latestMode)
	if cacheable && resolutionCache != nil {
		if entry, ok := resolutionCache[cacheKey]; ok {
			if entry.err != nil {
				if warnResolutionFailure != nil {
					warnResolutionFailure(action, entry.err)
				}
				return "", false, nil
			}
			return formatPinnedAction(refMeta.ActionPath, entry.commitSHA, entry.resolvedRef), true, nil
		}
	}

	// Call GetActionHashByVersion or GetBranchHash directly
	commitSHA, resolvedRef, err := resolveAndPin(refMeta, latestMode, resolveByVersion, resolveByBranch)

	// Cache the result (success or failure)
	if cacheable && resolutionCache != nil {
		resolutionCache[cacheKey] = actionResolutionCacheEntry{
			commitSHA:   commitSHA,
			resolvedRef: resolvedRef,
			err:         err,
		}
	}

	if err != nil {
		if warnResolutionFailure != nil {
			warnResolutionFailure(action, err)
		}
		return "", false, nil
	}

	return formatPinnedAction(refMeta.ActionPath, commitSHA, resolvedRef), true, nil
}
