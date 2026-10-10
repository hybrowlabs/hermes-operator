package cachescope

import (
	"strings"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func Parse(csv string) []string {
	var out []string
	seen := map[string]bool{}
	for _, p := range strings.Split(csv, ",") {
		p = strings.TrimSpace(p)
		if p == "" || seen[p] {
			continue
		}
		seen[p] = true
		out = append(out, p)
	}
	return out
}

func Options(secretNamespaces []string) cache.Options {
	if len(secretNamespaces) == 0 {
		return cache.Options{}
	}
	ns := make(map[string]cache.Config, len(secretNamespaces))
	for _, n := range secretNamespaces {
		ns[n] = cache.Config{}
	}
	return cache.Options{
		ByObject: map[client.Object]cache.ByObject{
			&corev1.Secret{}: {Namespaces: ns},
		},
	}
}
