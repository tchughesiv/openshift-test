package main

import (
	"fmt"
	"log"
	"os"
	"testing"

	bp "github.com/openshift/origin/pkg/cmd/server/bootstrappolicy"
	"github.com/openshift/origin/pkg/cmd/server/kubernetes/node"
	allocator "github.com/openshift/origin/pkg/security"
	admtesting "github.com/openshift/origin/pkg/security/admission/testing"
	securityapi "github.com/openshift/origin/pkg/security/apis/security"
	testutil "github.com/openshift/origin/test/util"
	testserver "github.com/openshift/origin/test/util/server"
	// appsv1beta1 "k8s.io/api/apps/v1beta1"
	// apiv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	//	"k8s.io/kubernetes/pkg/api/v1"
	"k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/pkg/apis/apps/v1beta1"
	"k8s.io/kubernetes/pkg/kubelet/cadvisor"
)

// command options/description reference ???
// https://github.com/openshift/origin/blob/release-3.6/pkg/cmd/cli/cli.go

func main() {
	defaultScc := "restricted"
	// defaultImage := "docker.io/centos:latest"
	var t *testing.T
	var sccopts []string
	var sccn *securityapi.SecurityContextConstraints
	_ = sccn

	if len(os.Args) > 1 {
		defaultScc = os.Args[len(os.Args)-1]
	}

	ns := admtesting.CreateNamespaceForTest()
	ns.Name = testutil.RandomNamespace("tmp")
	ns.Annotations[allocator.UIDRangeAnnotation] = "1000100000/10000"
	ns.Annotations[allocator.MCSAnnotation] = "s9:z0,z1"
	ns.Annotations[allocator.SupplementalGroupsAnnotation] = "1000100000/10000"

	groups, users := bp.GetBoostrapSCCAccess(ns.Name)
	bootstrappedConstraints := bp.GetBootstrapSecurityContextConstraints(groups, users)
	for _, v := range bootstrappedConstraints {
		sccopts = append(sccopts, v.Name)
		if v.Name == defaultScc {
			vtmp := v
			sccn = &vtmp
		}
	}

	if !contains(sccopts, defaultScc) {
		fmt.Printf("\n")
		fmt.Printf("%#v is not a valid scc. Must choose one of these:\n", defaultScc)
		for _, opt := range sccopts {
			fmt.Printf(" - %s\n", opt)
		}
		fmt.Printf("\n")
		os.Exit(1)
	}

	// How can supress the "startup" logs????
	etcdt := testutil.RequireEtcd(t)
	mconfig, nconfig, components, err := testserver.DefaultAllInOneOptions()
	checkErr(err)

	nodeconfig, err := node.BuildKubernetesNodeConfig(*nconfig, false, false)
	checkErr(err)
	// nodeconfig.Containerized = true

	kserver := nodeconfig.KubeletServer
	// kubeCfg := &kserver.KubeletConfiguration
	kubeDeps := nodeconfig.KubeletDeps
	if kubeDeps.CAdvisorInterface == nil {
		kubeDeps.CAdvisorInterface, err = cadvisor.New(uint(kserver.CAdvisorPort), kserver.ContainerRuntime, kserver.RootDirectory)
		checkErr(err)
	}
	// kubeCfg.ContainerRuntime = "docker"
	// kubeDeps.Recorder = record.NewFakeRecorder(100)

	// requires higher max user watches for file method...
	// sudo sysctl fs.inotify.max_user_watches=524288
	// make the change permanent, edit the file /etc/sysctl.conf and add the line to the end of the file
	// kubeCfg.PodManifestPath = kserver.RootDirectory + "/manifests"
	//pm := nconfig.PodManifestConfig
	//pm.Path = kubeCfg.PodManifestPath
	/*
		if _, err := os.Stat(kserver.RootDirectory); os.IsNotExist(err) {
			os.Mkdir(kserver.RootDirectory, 0755)
		}
		if _, err := os.Stat(kubeCfg.PodManifestPath); os.IsNotExist(err) {
			os.Mkdir(kubeCfg.PodManifestPath, 0750)
		}
	*/
	cfile, err := testserver.StartConfiguredAllInOne(mconfig, nconfig, components)
	// _, nconfig, cfile, err := testserver.StartTestAllInOne()
	checkErr(err)

	// provider, ns, err := scc.CreateProviderFromConstraint(ns.Name, ns, sccn, nodeconfig.Client)
	// checkErr(err)

	// !! can go straight k8s from here on out...
	//testpod := network.GetTestPod(defaultImage, "tcp", "tmp", "localhost", 12000)
	//tc := &testpod.Spec.Containers[0]
	//tc.SecurityContext, err = provider.CreateContainerSecurityContext(testpod, tc)
	//checkErr(err)

	//	v1Pod := &v1.Pod{}
	//	err = v1.Convert_api_Pod_To_v1_Pod(testpod, v1Pod, nil)
	//	checkErr(err)

	// !!! vendoring issues w/ kubelet packages
	// vendor/k8s.io/kubernetes/vendor/k8s.io/client-go/util/flowcontrol/throttle.go:59: undefined: ratelimit.Clock
	// try "startkubelet" instead? couldn't call it...
	//err = app.RunKubelet(kubeCfg, kubeDeps, false, true, kserver.DockershimRootDirectory)
	//checkErr(err)

	// use the current context in kubeconfig
	config, err := clientcmd.BuildConfigFromFlags("", cfile)
	if err != nil {
		panic(err.Error())
	}

	// create the clientset
	clientset, err := kubernetes.NewForConfig(config)
	checkErr(err)

	nsn := &v1.Namespace{}
	nsn.Name = ns.Name
	nsn.Annotations = ns.Annotations

	nsn, err = clientset.CoreV1().Namespaces().Create(nsn)
	checkErr(err)

	deploymentsClient := clientset.Apps().Deployments(nsn.Name)

	deployment := &v1beta1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name: "demo-deployment",
		},
		Spec: v1beta1.DeploymentSpec{
			Replicas: int32Ptr(1),
			Template: v1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": "demo",
					},
				},
				Spec: v1.PodSpec{
					Containers: []v1.Container{
						{
							Name:  "web",
							Image: "nginx:1.12",
							Ports: []v1.ContainerPort{
								{
									Name:          "http",
									Protocol:      v1.ProtocolTCP,
									ContainerPort: 80,
								},
							},
						},
					},
				},
			},
		},
	}

	// Create Deployment
	fmt.Println("Creating deployment...")
	result, err := deploymentsClient.Create(deployment)
	checkErr(err)
	fmt.Printf("Created deployment %q.\n", result.GetObjectMeta().GetName())

	//_, err = clientset.CoreV1().Pods(nsn.Name).Create(v1Pod)
	//checkErr(err)

	//pods, err := clientset.CoreV1().Pods(nsn.Name).List(metav1.ListOptions{})
	//checkErr(err)

	// Examples for error handling:
	// - Use helper functions like e.g. errors.IsNotFound()
	// - And/or cast to StatusError and use its properties like e.g. ErrStatus.Message
	_, err = clientset.CoreV1().Pods(nsn.Name).Get("web", metav1.GetOptions{})
	if errors.IsNotFound(err) {
		fmt.Printf("Pod not found\n")
	} else if statusError, isStatus := err.(*errors.StatusError); isStatus {
		fmt.Printf("Error getting pod %v\n", statusError.ErrStatus.Message)
	} else if err != nil {
		panic(err.Error())
	} else {
		fmt.Printf("Found pod\n")
	}

	//	time.Sleep(10 * time.Second)

	list, err := deploymentsClient.List(metav1.ListOptions{})
	checkErr(err)
	fmt.Printf("\n")
	for _, d := range list.Items {
		fmt.Printf(" * %s (%d replicas)\n", d.Name, *d.Spec.Replicas)
	}
	fmt.Printf("\n")

	err = os.RemoveAll(etcdt.DataDir)
	checkErr(err)

	// fmt.Printf("Using %#v scc...\n\n", provider.GetSCCName())
	// fmt.Printf("There are %d pods in the cluster\n\n", len(pods.Items))
}

func checkErr(err error) {
	if err != nil {
		log.Println(err)
	}
}

func contains(sccopts []string, defaultScc string) bool {
	for _, a := range sccopts {
		if a == defaultScc {
			return true
		}
	}
	return false
}

func int32Ptr(i int32) *int32 { return &i }

/*
	for {
		conn, err := net.DialTimeout("tcp", net.JoinHostPort("127.0.0.1", "10250"), 10*time.Second)
		if conn != nil {
			conn.Close()
			fmt.Printf("Server up on \n\n")
		}
		checkErr(err)
	}
*/
//kubeDeps.PodConfig.Sync()

/*
	k, err := kubelet.NewMainKubelet(kubeCfg, kubeDeps, true, kserver.DockershimRootDirectory)
	checkErr(err)

	podl, err := k.GetRunningPods()
	checkErr(err)
	podl = append(podl, v1Pod)

	fmt.Printf("%#v\n\n", podl[0])
	fmt.Printf("%#v\n\n", podl[0].Spec.Containers[0])
	fmt.Printf("%#v\n\n", podl[0].Spec.Containers[0].SecurityContext)

	kruntime := k.GetRuntime()
	imagelist, err := kruntime.ListImages()
	checkErr(err)
	fmt.Printf("%#v\n\n", imagelist)

	k.HandlePodAdditions(podl)
*/
/*
	k, err := app.CreateAndInitKubelet(kubeCfg, kubeDeps, true, kserver.DockershimRootDirectory)

	// kconfig := k.GetConfiguration()

		var secret []v1.Secret
		pi, err := kruntime.PullImage(container.ImageSpec{
			Image: v1Pod.Spec.Containers[0].Image,
		}, secret)
		checkErr(err)
		fmt.Printf("\n")
		fmt.Printf("%#v\n\n", pi)

	imagelist, err := kruntime.ListImages()
	checkErr(err)
	fmt.Printf("%#v\n\n", imagelist)


	var updates <-chan kubetypes.PodUpdate
		updates <- kubetypes.PodUpdate{
			Pods:   podl,
			Op:     kubetypes.ADD,
			Source: kubetypes.AllSource,
		}

	runresult, err := k.RunOnce(updates)
	checkErr(err)
	fmt.Printf("\n%#v\n\n", runresult)

	fmt.Printf("%#v\n\n", k.GetActivePods())
*/
