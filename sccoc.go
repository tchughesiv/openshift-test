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
	"k8s.io/kubernetes/pkg/kubelet"
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
	// dockerVersion := "v1.12.6"
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
	clusterAdminKubeClientset, err := testutil.GetClusterAdminKubeClient(testutil.KubeConfigPath())
	checkErr(err)
	nodeconfig, err := node.BuildKubernetesNodeConfig(*nconfig, false, false)
	checkErr(err)
	err = os.RemoveAll(etcdt.DataDir)
	checkErr(err)

	// reference Admit function vendor/github.com/openshift/origin/pkg/security/admission/admission.go
	fmt.Printf("\n")
	provider, ns, err := scc.CreateProviderFromConstraint(ns.Name, ns, sccn, clusterAdminKubeClientset)
	checkErr(err)
	// testis := testgen.MockImageStream("centos", "docker.io/centos", map[string]string{"latest": "latest"})
	// testpod := testutil.CreatePodFromImage(testis, "latest", ns.Name)
	testpod := network.GetTestPod(defaultImage, "tcp", "tmp", "localhost", 12000)

	tc := &testpod.Spec.Containers[0]
	tc.SecurityContext, err = provider.CreateContainerSecurityContext(testpod, tc)
	checkErr(err)

	fmt.Printf("\n%#v\n\n", tc.SecurityContext)
	// fmt.Printf("%#v\n\n", tc.SecurityContext.Capabilities)
	// fmt.Printf("%#v\n\n", tc.SecurityContext.SELinuxOptions.Level)
	// fmt.Printf("%#v\n\n", dcfg.Endpoint)
	fmt.Printf("Using %#v scc...\n\n", provider.GetSCCName())
	// fmt.Printf("%#v\n\n", dcfg.Endpoint)

	// vendoring issues w/ kubelet packages
	// vendor/k8s.io/kubernetes/vendor/k8s.io/client-go/util/flowcontrol/throttle.go:59: undefined: ratelimit.Clock

	// SOOO, instead use vendor/github.com/openshift/origin/pkg/cmd/server/start/start_node.go to build a NodeConfig
	// which has everything you need below to GenerateRunContainerOptions
	// nodeConfig, configFile, err := resolveNodeConfig()
	// nconfig.MasterKubeConfig

	// start.Run()

	kserver := nodeconfig.KubeletServer
	kubeDeps := nodeconfig.KubeletDeps
	kubeCfg := kserver.KubeletConfiguration
	kubelet.KubeletBootstrap
	k, err := kubelet.NewMainKubelet(kubeCfg, kubeDeps, true, kserver.DockershimRootDirectory)
	//	k, err := kubelet.NewMainKubelet(nodeconfig.KubeletConfiguration, kubeDeps, true, nodeconfig.DockershimRootDirectory)
	//	checkErr(err)
	//	rco, _, err := k.GenerateRunContainerOptions(testpod, tc, nodeconfig.Address)
	//	checkErr(err)
	fmt.Printf("%#v\n\n", kubeCfg.Address)
	//	fmt.Printf("%#v\n\n", rco)

	//	node.BuildKubernetesNodeConfig()

	// ?? reference for container runtime -
	// vendor/github.com/openshift/origin/vendor/k8s.io/kubernetes/pkg/kubelet/kubelet.go
	// vendor/github.com/openshift/origin/vendor/k8s.io/kubernetes/pkg/kubectl/run_test.go
	// kubectl run reference: https://github.com/openshift/kubernetes/blob/openshift-1.6-20170501/pkg/kubectl/run_test.go

	// dockerRun(tc.Image, dockerVersion)
}

// INSTEAD ... possible reuse functions from here:
// /home/tohughes/Documents/Workspace/go_path/src/github.com/tchughesiv/sccoc/vendor/github.com/openshift/source-to-image/pkg/docker/docker.go
func dockerRun(image string, dockerVersion string) {
	/*
		ctx := context.Background()
		cli, err := dockerapi.NewClient(dockerapi.DefaultDockerHost, dockerVersion, nil, nil)
		if err != nil {
			panic(err)
		}

		// docker.NewEngineAPIClient()
		ilist, err := cli.ImageList(ctx, dockertypes.ImageListOptions{MatchName: image})
		if len(ilist) == 0 {
			iresp, err := cli.ImagePull(ctx, image, dockertypes.ImagePullOptions{})
			if err != nil {
				panic(err)
			}
			// how do i pretty up the iresp to stdout?
			io.Copy(os.Stdout, iresp)
		}

		resp, err := cli.ContainerCreate(ctx, &dockercontainer.Config{
			Image: image,
			Cmd:   []string{"echo", "hello world"},
		}, nil, nil, "")
		if err != nil {
			panic(err)
		}

		if err := cli.ContainerStart(ctx, resp.ID); err != nil {
			panic(err)
		}

		_, err = cli.ContainerWait(ctx, resp.ID)
		if err != nil {
			panic(err)
		}

		out, err := cli.ContainerLogs(ctx, resp.ID, dockertypes.ContainerLogsOptions{ShowStdout: true})
		if err != nil {
			panic(err)
		}
		io.Copy(os.Stdout, out)
	*/
}
