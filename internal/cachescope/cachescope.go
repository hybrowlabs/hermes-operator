package cachescope

import (
	"fmt"
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

func Allows(secretNamespaces []string, namespace string) bool {
	if len(secretNamespaces) == 0 {
		return true
	}
	for _, n := range secretNamespaces {
		if n == namespace {
			return true
		}
	}
	return false
}

func NotInScopeMessage(namespace string, secretNamespaces []string) string {
	return fmt.Sprintf("namespace %q is not in --secret-namespaces (%s): this operator instance cannot read Secrets there, so Secret references cannot be checked or reconciled; create the instance in a listed namespace or widen the flag and the Secret Role",
		namespace, strings.Join(secretNamespaces, ","))
}
