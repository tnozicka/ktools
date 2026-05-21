package collect

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"

	clnaming "github.com/tnozicka/k8s-controller-lib/pkg/naming"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	corev1client "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/klog/v2"
)

func retrieveContainerLogs(
	ctx context.Context,
	podClient corev1client.PodInterface,
	destinationPath string,
	podName string,
	logOptions *corev1.PodLogOptions,
) error {
	dest, err := os.OpenFile(destinationPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0666)
	if err != nil {
		return fmt.Errorf("can't open file %q: %w", destinationPath, err)
	}
	defer func() {
		closeErr := dest.Close()
		if closeErr != nil {
			klog.ErrorS(closeErr, "can't close file", "Path", destinationPath)
		}
	}()

	logsReq := podClient.GetLogs(podName, logOptions)
	readCloser, err := logsReq.Stream(ctx)
	if err != nil {
		return fmt.Errorf("can't create log stream: %w", err)
	}
	defer func() {
		closeErr := readCloser.Close()
		if closeErr != nil {
			klog.ErrorS(
				closeErr, "can't close log stream",
				"Path", destinationPath,
				"Pod", podName,
				"Container", logOptions.Container,
			)
		}
	}()

	_, err = io.Copy(dest, readCloser)
	if err != nil {
		return fmt.Errorf("can't read logs: %w", err)
	}

	klog.V(4).InfoS(
		"Wrote container logs",
		"Path", destinationPath,
		"Pod", podName,
		"Container", logOptions.Container,
	)

	return nil
}

func (c *Collector) collectContainerLogs(
	ctx context.Context,
	logsDir string,
	podMeta *metav1.ObjectMeta,
	podCSs []corev1.ContainerStatus,
	containerName string,
) error {
	var err error

	i := slices.IndexFunc(
		podCSs,
		func(s corev1.ContainerStatus) bool {
			return s.Name == containerName
		},
	)
	if i < 0 {
		klog.InfoS("Kubelet has not reported status for container yet", "Pod", clnaming.ObjNN(podMeta), "Container", containerName)
		return nil
	}
	cs := podCSs[i]

	var limitBytes *int64
	if c.logsLimitBytes > 0 {
		limitBytes = new(c.logsLimitBytes)
	}
	logOptions := &corev1.PodLogOptions{
		Container:  containerName,
		Timestamps: true,
		Follow:     false,
		LimitBytes: limitBytes,
	}

	if cs.State.Running != nil {
		logOptions.Previous = false
		err = retrieveContainerLogs(
			ctx,
			c.coreClient.Pods(podMeta.Namespace),
			filepath.Join(logsDir, containerName+".current"),
			podMeta.Name,
			logOptions,
		)
		if err != nil {
			return fmt.Errorf("can't retrieve pod logs for container %q in pod %q: %w", containerName, clnaming.ObjNN(podMeta), err)
		}
	}

	if cs.LastTerminationState.Terminated != nil {
		logOptions.Previous = true
		err = retrieveContainerLogs(
			ctx,
			c.coreClient.Pods(podMeta.Namespace),
			filepath.Join(logsDir, containerName+".previous"),
			podMeta.Name,
			logOptions,
		)
		if err != nil {
			return fmt.Errorf("can't retrieve previous pod logs for container %q in pod %q: %w", containerName, clnaming.ObjNN(podMeta), err)
		}
	}

	return nil
}

func (c *Collector) collectPod(ctx context.Context, resourceInfo *ResourceInfo, u *unstructured.Unstructured) error {
	pod := &corev1.Pod{}
	err := runtime.DefaultUnstructuredConverter.FromUnstructured(u.Object, pod)
	if err != nil {
		return fmt.Errorf("can't convert unstructured into a pod: %w", err)
	}

	err = c.writeObject(ctx, resourceInfo, pod)
	if err != nil {
		return fmt.Errorf("can't write pod %q: %w", clnaming.ObjNN(pod), err)
	}

	if c.collectLogs {
		resourceDir, err := c.getResourceLocation(pod, resourceInfo)
		if err != nil {
			return fmt.Errorf("can't get resourceDir: %q", err)
		}

		logsDir := filepath.Join(resourceDir, pod.GetName())

		err = os.MkdirAll(logsDir, 0770)
		if err != nil {
			return fmt.Errorf("can't create logs dir %q: %w", logsDir, err)
		}

		for _, container := range pod.Spec.InitContainers {
			err = c.collectContainerLogs(ctx, logsDir, &pod.ObjectMeta, pod.Status.InitContainerStatuses, container.Name)
			if err != nil {
				return fmt.Errorf("can't collect logs for init container %q in pod %q: %w", container.Name, clnaming.ObjNN(pod), err)
			}
		}

		for _, container := range pod.Spec.Containers {
			err = c.collectContainerLogs(ctx, logsDir, &pod.ObjectMeta, pod.Status.ContainerStatuses, container.Name)
			if err != nil {
				return fmt.Errorf("can't collect logs for container %q in pod %q: %w", container.Name, clnaming.ObjNN(pod), err)
			}
		}

		for _, container := range pod.Spec.EphemeralContainers {
			err = c.collectContainerLogs(ctx, logsDir, &pod.ObjectMeta, pod.Status.EphemeralContainerStatuses, container.Name)
			if err != nil {
				return fmt.Errorf("can't collect logs for ephemeral container %q in pod %q: %w", container.Name, clnaming.ObjNN(pod), err)
			}
		}
	}

	return nil
}
