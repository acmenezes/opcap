package operator

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// GetPackageManifests

// CreateNamespace
func CreateNamespace(c *kubernetes.Clientset) {

	OpcapNs := corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{Name: "opcap"},
		Spec:       corev1.NamespaceSpec{Finalizers: []corev1.FinalizerName{"kubernetes"}},
	}
	ns, err := c.CoreV1().Namespaces().Create(context.Background(), &OpcapNs, metav1.CreateOptions{})
	if err != nil {
		fmt.Printf("Unable to create Namespace: %s\n", err)
	}
	fmt.Printf("Created Namespace: %s\n", ns.ObjectMeta.Name)
}

// CreateOperatorGroup
func CreateOperatorGroup(c *kubernetes.Clientset) {

}

// CreateSubscription
func CreateSubscription() {

}

// ApproveInstallPlan
func ApproveInstallPlan() {

}

// GetCSVStatus
func GetCSVStatus() {

}

// OperatorCleanUp
func OperatorCleanUp() {

}

// // DeleteSubscription

// // DeleteOperatorGroup

// // DeleteNamespace

// Installer:
// 1. openshift-install create cluster --install-config myfile.yaml
// 2. wait for install to complete
//
// ---- for each operator on queue ----
//
// Bundle:
// 3. run bundle (create operator group, subscription, approve install plan)
// 4. wait for operator to be ready - check CSV to be ready
//
// CR:
// 5. create CR
// 6. wait for CR to be ready
//
// CAPABILITY:
// 7. run tests (split multiple times in each of the 5 levels)
// 8. retrieve data
// 9. repeat until finish
//
// REPORT:
// 10. generate and publish report
//
// CLEAN UP OPERATOR:
// 11. clean up operator and operand
//
// ---- go next until queue is complete ---
//
// DELETE CLUSTER:
// 12. clean up cluster and exit
