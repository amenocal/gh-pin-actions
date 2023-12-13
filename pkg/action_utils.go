package pkg

import (
	"fmt"
	"strings"
)

func SplitActionString(action string, delimiter string) (string, string, error) {
	actionSplit := strings.Split(action, delimiter)
	if len(actionSplit) < 2 {
		return "", "", fmt.Errorf("invalid action format: %s", action)
	}
	repoWithOwner := actionSplit[0]
	branchOrVersion := actionSplit[1]
	return repoWithOwner, branchOrVersion, nil
}
