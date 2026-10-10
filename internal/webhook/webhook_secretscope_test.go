package webhook

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"

	hermesv1 "github.com/paperclipinc/hermes-operator/api/v1"
)

func scopedValidator(t *testing.T, scope []string, getCalls *int) *HermesInstanceValidator {
	t.Helper()
	c := fake.NewClientBuilder().WithScheme(newScheme(t)).WithInterceptorFuncs(interceptor.Funcs{
		Get: func(_ context.Context, _ client.WithWatch, _ client.ObjectKey, obj client.Object, _ ...client.GetOption) error {
			if _, ok := obj.(*corev1.Secret); ok {
				*getCalls++
			}
			return errors.New("unknown namespace for the cache")
		},
	}).Build()
	return &HermesInstanceValidator{Client: c, SecretNamespaces: scope}
}

func telegramInstance(ns string) *hermesv1.HermesInstance {
	return &hermesv1.HermesInstance{
		ObjectMeta: metav1.ObjectMeta{Name: "demo", Namespace: ns},
		Spec: hermesv1.HermesInstanceSpec{
			Image:   hermesv1.ImageSpec{Repository: "x"},
			Storage: hermesv1.StorageSpec{Persistence: hermesv1.PersistenceSpec{Size: "1Gi"}},
			Gateways: hermesv1.GatewaysSpec{
				Telegram: hermesv1.TelegramGatewaySpec{
					Enabled: Ptr(true),
					BotTokenSecretRef: &corev1.SecretKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{Name: "tg"},
						Key:                  "token",
					},
				},
			},
		},
	}
}

func TestSecretScope_UnlistedNamespaceIsNotDeniedForCacheReasons(t *testing.T) {
	calls := 0
	v := scopedValidator(t, []string{"hermes-ck"}, &calls)
	warns, err := v.ValidateCreate(context.Background(), telegramInstance("paperclip"))
	require.NoError(t, err, "unlisted namespace must not be denied by a cache error")
	assert.Zero(t, calls, "no Secret read may be attempted outside the scoped namespaces")
	require.NotEmpty(t, warns)
	assert.Contains(t, joined(warns), "--secret-namespaces")
}

func TestSecretScope_ListedNamespaceStillChecksSecrets(t *testing.T) {
	calls := 0
	v := scopedValidator(t, []string{"hermes-ck"}, &calls)
	_, err := v.ValidateCreate(context.Background(), telegramInstance("hermes-ck"))
	assert.Error(t, err, "real lookup errors in a listed namespace are still surfaced")
	assert.Positive(t, calls)
}

func TestSecretScope_EmptyScopeKeepsExistingBehaviour(t *testing.T) {
	calls := 0
	v := scopedValidator(t, nil, &calls)
	_, err := v.ValidateCreate(context.Background(), telegramInstance("anything"))
	assert.Error(t, err)
	assert.Positive(t, calls)
}

func joined(ws []string) string {
	s := ""
	for _, w := range ws {
		s += w + "\n"
	}
	return s
}
