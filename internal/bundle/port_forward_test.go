package bundle

import (
	. "github.com/onsi/ginkgo/v2/dsl/core"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

var _ = Describe("Contalog Port Forward", func() {
	var scheme *runtime.Scheme

	BeforeEach(func() {
		scheme = runtime.NewScheme()
	})
	When("Calling the function with catalog name", func() {

		It("should return catalog's pod name", func() {
			catalogPod := &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name: "certified-operators-nhwyd",
				},
			}

			fakeClientBuilder := fake.NewClientBuilder().WithScheme(scheme)

			fakeClient := fakeClientBuilder.WithObjects([]client.Object{catalogPod}...).Build()

			podName, err := getGrpcPodNameForCatalog("certified-operators", fakeClient)

			Expect(err).ToNot(HaveOccurred())

			Expect(podName).To(Equal("certified-operators-nhwyd"))
		})

	})
})
