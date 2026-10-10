package controller

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"

	hermesv1 "github.com/paperclipinc/hermes-operator/api/v1"
)

func scopedReconciler(t *testing.T, scope []string, calls *int) *HermesInstanceReconciler {
	t.Helper()
	s := runtime.NewScheme()
	require.NoError(t, clientgoscheme.AddToScheme(s))
	require.NoError(t, hermesv1.AddToScheme(s))
	c := fake.NewClientBuilder().WithScheme(s).WithInterceptorFuncs(interceptor.Funcs{
		Get: func(_ context.Context, _ client.WithWatch, _ client.ObjectKey, obj client.Object, _ ...client.GetOption) error {
			if _, ok := obj.(*corev1.Secret); ok {
				*calls++
			}
			return errors.New("unknown namespace for the cache")
		},
	}).Build()
	return &HermesInstanceReconciler{Client: c, Scheme: s, SecretNamespaces: scope}
}

func TestReconcileSecret_UnlistedNamespaceFailsWithExplicitMessage(t *testing.T) {
	calls := 0
	r := scopedReconciler(t, []string{"hermes-ck"}, &calls)
	inst := &hermesv1.HermesInstance{ObjectMeta: metav1.ObjectMeta{Name: "x", Namespace: "paperclip"}}
	err := r.reconcileSecret(context.Background(), inst)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "--secret-namespaces")
	assert.Contains(t, err.Error(), "paperclip")
	assert.Zero(t, calls, "must not attempt a cached Secret read outside scope")
}

func TestReconcileSecret_EmptyScopeStillReadsSecret(t *testing.T) {
	calls := 0
	r := scopedReconciler(t, nil, &calls)
	inst := &hermesv1.HermesInstance{ObjectMeta: metav1.ObjectMeta{Name: "x", Namespace: "paperclip"}}
	_ = r.reconcileSecret(context.Background(), inst)
	assert.Positive(t, calls)
}
