package bundle

import (
	"context"
	"fmt"

	// "log"
	"net/http"
	"strings"

	// "os/exec"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/rest"

	// "github.com/goccy/kpoward"
	// "k8s.io/client-go/transport"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"

	"net/url"
	"os"
	"os/signal"

	// "sync"
	"syscall"

	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/tools/portforward"
	"k8s.io/client-go/transport/spdy"
)

func getk8sClient() (client.Client, error) {

	k8sconfig, err := config.GetConfig()
	if err != nil {
		return nil, err
	}

	c, err := client.New(k8sconfig, client.Options{})
	if err != nil {
		return nil, err
	}

	return c, nil
}

func getGrpcPodNameForCatalog(catalog string, c client.Client) (string, error) {

	opts := &client.ListOptions{
		Namespace: "openshift-marketplace",
	}

	PodList := &corev1.PodList{}

	err := c.List(context.TODO(), PodList, opts)
	if err != nil {
		return "", err
	}

	for _, pod := range PodList.Items {

		if strings.Contains(pod.ObjectMeta.Name, catalog) && pod.Status.Phase == "Running" {
			fmt.Println(pod.ObjectMeta.Name)
			return pod.ObjectMeta.Name, nil
		}
	}

	return "", fmt.Errorf("no %s pod found", catalog)
}

type PortForwardAPodRequest struct {
	// RestConfig is the kubernetes config
	RestConfig *rest.Config
	// Pod is the selected pod for this port forwarding
	PodName      string
	PodNamespace string
	// LocalPort is the local port that will be selected to expose the PodPort
	LocalPort int
	// PodPort is the target port for the pod
	PodPort int
	// Steams configures where to write or read input from
	Streams genericclioptions.IOStreams
	// StopCh is the channel used to manage the port forward lifecycle
	StopCh <-chan struct{}
	// ReadyCh communicates when the tunnel is ready to receive traffic
	ReadyCh chan struct{}
}

func portForward(podName string, k8sconfig *rest.Config) error {
	// var wg sync.WaitGroup
	// wg.Add(1)

	stopCh := make(chan struct{})
	readyCh := make(chan struct{})
	stream := genericclioptions.IOStreams{
		In:     os.Stdin,
		Out:    os.Stdout,
		ErrOut: os.Stderr,
	}

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigs
		fmt.Println("Bye...")
		close(stopCh)
		// wg.Done()
	}()

	req := &PortForwardAPodRequest{
		RestConfig:   k8sconfig,
		PodName:      podName,
		PodNamespace: "openshift-marketplace",
		LocalPort:    50051,
		PodPort:      50051,
		Streams:      stream,
		StopCh:       stopCh,
		ReadyCh:      readyCh,
	}

	go func() {
		_, err := PortForwardAPod(req)
		if err != nil {
			panic(err)
		}
		// wg.Done()
	}()

	for msg := range readyCh {
		fmt.Print(msg)
		break
	}

	fmt.Println("Port forwarding is ready to get traffic. Have fun!")
	// wg.Wait()

	return nil
}

func PortForwardAPod(req *PortForwardAPodRequest) (*portforward.PortForwarder, error) {
	path := fmt.Sprintf("%s/api/v1/namespaces/%s/pods/%s/portforward",
		req.RestConfig, req.PodNamespace, req.PodName)
	hostIP := strings.TrimLeft(req.RestConfig.Host, "htps:/")

	transport, upgrader, err := spdy.RoundTripperFor(req.RestConfig)
	if err != nil {
		return nil, err
	}

	dialer := spdy.NewDialer(upgrader, &http.Client{Transport: transport}, http.MethodPost, &url.URL{Scheme: "https", Path: path, Host: hostIP})
	fw, err := portforward.New(dialer, []string{fmt.Sprintf("%d:%d", req.LocalPort, req.PodPort)}, req.StopCh, req.ReadyCh, req.Streams.Out, req.Streams.ErrOut)
	fw.ForwardPorts()
	if err != nil {
		fmt.Println("fp error")
		panic(err)
	}

	return fw, nil
}

// ---------------------------- os/exec
// func catalogPortForward(podName string, catalog string) error {

// 	args := []string{"oc port-forward -n openshift-martketplace --address localhost pod/" + podName + " 50051:50051"}
// 	cmd := exec.Command("bash", args...)
// 	err := cmd.Start()
// 	if err != nil {
// 		log.Fatal(err)
// 	}
// 	log.Printf("Waiting for command to finish...")
// 	log.Printf("Process id is %v", cmd.Process.Pid)
// 	err = cmd.Wait()
// 	log.Printf("Command finished with error, now restarting: %v", err)
// 	// if err := cmd.Run(); err != nil {
// 	// 	log.Fatal(err)
// 	// }
// 	fmt.Println("before catalog start")
// 	cmd.Start()

// 	return nil
// }

// ------------------------------ remove go routine

// _, err := PortForwardAPod(req)
// fmt.Printf("end of pf")
// if err != nil {
// 	panic(err)
// }

// fmt.Println("before range")
// go func() {
// 	for msg := range readyCh {
// 		fmt.Print(msg)
// 		break
// 	}
// }()

// select {
// case <-readyCh:
// 	fmt.Println("in readyCh")
// 	break
// }

// ---------------------------------------- kpoward

// func portForward(podName string, k8sconfig *rest.Config) {

// 	var (
// 		restCfg = k8sconfig
// 		targetPodName = podName
// 		targetPort = 50051
// 	)

// 	kpow := kpoward.New(restCfg, targetPodName, uint16(targetPort))
// 	fmt.Println("new kpow created")
// 	if err := kpow.Run(context.Background(), func(ctx context.Context, localPort uint16) error {
// 		fmt.Println("kpow running")
// 		log.Printf("localPort: %d", localPort)
// 		resp, err := http.Get(fmt.Sprintf("http://localhost:%d", localPort))
// 		if err != nil {
// 			return err
// 		}
// 		defer resp.Body.Close()
// 		return nil
// 	}); err != nil {
// 		fmt.Errorf("unable to get port")
// 	}
// }
