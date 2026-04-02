package kubetypes

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

type Object interface {
	runtime.Object
	metav1.Object
}

func ObjectAccessor(obj any) (Object, error) {
	res, ok := obj.(Object)
	if !ok {
		return nil, fmt.Errorf("object %T is not an Object", obj)
	}

	return res, nil
}

func GetObjectGVK(obj runtime.Object, s *runtime.Scheme) (schema.GroupVersionKind, error) {
	gvks, _, err := s.ObjectKinds(obj)
	if err != nil {
		return schema.GroupVersionKind{}, fmt.Errorf("can't get kind for object %T", obj)
	}

	if len(gvks) == 0 {
		return schema.GroupVersionKind{}, fmt.Errorf("no kind is registered for object %T", obj)
	}

	return gvks[0], nil
}

func GetObjectGVKORUnknown(obj runtime.Object, s *runtime.Scheme) schema.GroupVersionKind {
	gvk, err := GetObjectGVK(obj, s)
	if err != nil {
		return schema.GroupVersionKind{
			Group:   "<unknown>",
			Version: "<unknown>",
			Kind:    "<unknown>",
		}
	}

	return gvk
}

type ObjectGetter[T Object] interface {
	Get(ctx context.Context, name string, opts metav1.GetOptions) (T, error)
}

type ObjectDeleter interface {
	Delete(ctx context.Context, name string, opts metav1.DeleteOptions) error
}

type GetterDeleter[T Object] interface {
	ObjectGetter[T]
	ObjectDeleter
}
