package main

import (
	"fmt"
	"log"
	"os"
	"testing"

	bp "github.com/openshift/origin/pkg/cmd/server/bootstrappolicy"
	"github.com/openshift/origin/pkg/cmd/server/kubernetes/node"
	"github.com/openshift/origin/pkg/diagnostics/network"
	allocator "github.com/openshift/origin/pkg/security"
	admtesting "github.com/openshift/origin/pkg/security/admission/testing"
	securityapi "github.com/openshift/origin/pkg/security/apis/security"
	"github.com/openshift/origin/pkg/security/scc"
	testutil "github.com/openshift/origin/test/util"
	testserver "github.com/openshift/origin/test/util/server"
	"k8s.io/client-go/tools/record"
	"k8s.io/kubernetes/pkg/api/v1"
	"k8s.io/kubernetes/pkg/kubelet"
	"k8s.io/kubernetes/pkg/kubelet/cadvisor"
)

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

// command options/description reference:
// https://github.com/openshift/origin/blob/release-3.6/pkg/cmd/cli/cli.go

func main() {
	defaultScc := "restricted"
	defaultImage := "docker.io/centos:latest"
	var t *testing.T
	var sccopts []string
	var sccn *securityapi.SecurityContextConstraints

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
	_, nconfig, _, err := testserver.DefaultAllInOneOptions()
	checkErr(err)
	nodeconfig, err := node.BuildKubernetesNodeConfig(*nconfig, false, false)
	checkErr(err)
	provider, ns, err := scc.CreateProviderFromConstraint(ns.Name, ns, sccn, nodeconfig.Client)
	checkErr(err)
	err = os.RemoveAll(etcdt.DataDir)
	checkErr(err)

	// !! can go straight k8s from here on out...
	testpod := network.GetTestPod(defaultImage, "tcp", "tmp", "localhost", 12000)
	tc := &testpod.Spec.Containers[0]
	tc.SecurityContext, err = provider.CreateContainerSecurityContext(testpod, tc)
	checkErr(err)
	v1Pod := &v1.Pod{}
	err = v1.Convert_api_Pod_To_v1_Pod(testpod, v1Pod, nil)
	checkErr(err)

	// !!! vendoring issues w/ kubelet packages
	// vendor/k8s.io/kubernetes/vendor/k8s.io/client-go/util/flowcontrol/throttle.go:59: undefined: ratelimit.Clock

	kserver := nodeconfig.KubeletServer
	kubeCfg := &kserver.KubeletConfiguration
	// kubeCfg.ContainerRuntime = "docker"
	kubeDeps := nodeconfig.KubeletDeps
	kubeDeps.Recorder = record.NewFakeRecorder(100)
	if kubeDeps.CAdvisorInterface == nil {
		kubeDeps.CAdvisorInterface, err = cadvisor.New(uint(kserver.CAdvisorPort), kserver.ContainerRuntime, kserver.RootDirectory)
		checkErr(err)
	}
	k, err := kubelet.NewMainKubelet(kubeCfg, kubeDeps, true, kserver.DockershimRootDirectory)
	checkErr(err)

	fmt.Printf("\n")

	// klConfig := k.GetConfiguration()
	// fmt.Printf("%#v\n\n", klConfig)

	runtime := k.GetRuntime()
	// fmt.Printf("%#v\n\n", runtime)

	/*
		var secret []v1.Secret
		pi, err := runtime.PullImage(container.ImageSpec{
			Image: v1Pod.Spec.Containers[0].Image,
		}, secret)
		checkErr(err)
		fmt.Printf("\n")
		fmt.Printf("%#v\n\n", pi)
	*/

	imagelist, err := runtime.ListImages()
	checkErr(err)
	fmt.Printf("%#v\n\n", imagelist)

	podl, err := k.GetRunningPods()
	checkErr(err)
	fmt.Printf("%#v\n\n", podl)

	podl = append(podl, v1Pod)
	fmt.Printf("%#v\n\n", podl)
	fmt.Printf("%#v\n\n", podl[0])
	fmt.Printf("%#v\n\n", podl[0].Spec.Containers[0])
	// k.HandlePodAdditions(podl)

	/*
		klclient := k.GetKubeClient()
		di := klclient.Apps().Deployments(ns.Name)
		dl, err := di.List(metav1.ListOptions{})
		checkErr(err)
		fmt.Printf("%#v\n\n", dl)
	*/

	// ?? reference for container runtime -
	// vendor/github.com/openshift/origin/vendor/k8s.io/kubernetes/pkg/kubelet/kubelet.go
	// vendor/github.com/openshift/origin/vendor/k8s.io/kubernetes/pkg/kubectl/run_test.go
	// kubectl run reference: https://github.com/openshift/kubernetes/blob/openshift-1.6-20170501/pkg/kubectl/run_test.go
	// dockertools.NewDockerManager()
	// dockerRun(tc.Image, dockerVersion)

	// fmt.Printf("%#v\n\n", podl[0].Spec.Containers[0].Name)
	fmt.Printf("%#v\n\n", podl[0].Spec.Containers[0].SecurityContext)
	fmt.Printf("Using %#v scc...\n\n", provider.GetSCCName())
}
