package collect

import (
	"context"
	"fmt"

	clnaming "github.com/tnozicka/k8s-controller-lib/pkg/naming"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
)

func isPublicSecretKey(key string) bool {
	switch key {
	case "ca.crt", "tls.crt", "service-ca.crt":
		return true
	default:
		return false
	}
}

func (c *Collector) collectSecret(ctx context.Context, resourceInfo *ResourceInfo, u *unstructured.Unstructured) error {
	secret := &corev1.Secret{}
	err := runtime.DefaultUnstructuredConverter.FromUnstructured(u.Object, secret)
	if err != nil {
		return fmt.Errorf("can't convert unstructured into a secret: %w", err)
	}

	for k := range secret.Data {
		if !isPublicSecretKey(k) {
			secret.Data[k] = []byte("<redacted>")
		}
	}

	err = c.writeObject(ctx, resourceInfo, secret)
	if err != nil {
		return fmt.Errorf("can't write secret %q: %w", clnaming.ObjNN(secret), err)
	}

	return nil
}
