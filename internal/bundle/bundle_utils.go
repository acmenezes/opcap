package bundle

import (
	"context"
	"fmt"
	"log"
	"strings"

	"os/exec"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

func getGrpcPodNameForCatalog(catalog string) (string, error) {

	k8sconfig, err := config.GetConfig()
	if err != nil {
		return "", err
	}

	c, err := client.New(k8sconfig, client.Options{})
	if err != nil {
		return "", err
	}

	opts := &client.ListOptions{
		Namespace: "openshift-marketplace",
	}

	PodList := &corev1.PodList{}

	err = c.List(context.TODO(), PodList, opts)
	if err != nil {
		return "", err
	}

	for _, pod := range PodList.Items {

		if strings.Contains(pod.ObjectMeta.Name, catalog) && pod.Status.Phase == "Running" {
			return pod.ObjectMeta.Name, nil
		}
	}

	return "", fmt.Errorf("No %s pod found.", catalog)
}

// os/exec
func catalogPortForward(podName string) error {

	cmd := exec.Command("oc port-forward +")
	if err := cmd.Run(); err != nil {
		log.Fatal(err)
	}

	return nil
}
