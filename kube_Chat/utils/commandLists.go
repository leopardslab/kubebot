package utils

import "sort"

type Category struct {
	categories []string
}

//const for command categories supported by Kubectl do not change. Stored in sorted manner.

var list = []string{"alpha", "annotate", "api-resources", "api-versions", "apply", "attach", "auth", "autoscale", "certificate",
	"cluster-info", "completion", "config", "cordon", "cp", "create", "debug", "delete", "describe", "diff", "drain", "edit", "exec",
	"explain", "expose", "get", "kustomize", "label", "logs", "patch", "plugin", "port-forward", "proxy", "replace", "rollout", "run",
	"scale", "set", "taint", "top", "uncordon", "version", "wait"}

func GetCategories() *Category {
	return &Category{
		categories: list,
	}
}

func IsCategory(cat string) bool {
	//sor.Strings(list)
	res := sort.SearchStrings(list, cat)
	if list[res] == cat {
		return true
	}
	return false
}
