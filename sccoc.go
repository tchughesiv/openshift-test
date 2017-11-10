package main

import (
	"fmt"
	"log"
	"os"
	"testing"

	bp "github.com/openshift/origin/pkg/cmd/server/bootstrappolicy"
	"github.com/openshift/origin/pkg/generate/app"
	imageapi "github.com/openshift/origin/pkg/image/apis/image"
	projectapi "github.com/openshift/origin/pkg/project/apis/project"
	allocator "github.com/openshift/origin/pkg/security"
	admtesting "github.com/openshift/origin/pkg/security/admission/testing"
	securityapi "github.com/openshift/origin/pkg/security/apis/security"
	testutil "github.com/openshift/origin/test/util"
	testserver "github.com/openshift/origin/test/util/server"
)

// command options/description reference ???
// https://github.com/openshift/origin/blob/release-3.6/pkg/cmd/cli/cli.go

func main() {
	defaultScc := "restricted"
	defaultImage := "centos"
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
	// defer testutil.DumpEtcdOnFailure(t)
	mconfig, nconfig, components, err := testserver.DefaultAllInOneOptions()
	checkErr(err)

	/*
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
	*/

	kconfig, err := testserver.StartConfiguredAllInOne(mconfig, nconfig, components)
	cac, err := testutil.GetClusterAdminClient(kconfig)
	checkErr(err)

	// clusterAdminClientConfig, err := testutil.GetClusterAdminClientConfig(kconfig)
	// checkErr(err)

	// !! can go straight k8s from here on out...
	// testpod := network.GetTestPod(defaultImage, "tcp", "tmp", "localhost", 12000)
	// tc := testpod.Spec.Containers[]
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

	project := &projectapi.Project{}
	project.Name = ns.Name
	project.Annotations = ns.Annotations

	projcl := cac.Projects()
	proj, err := projcl.Create(project)
	checkErr(err)

	dccl := cac.DeploymentConfigs(proj.Name)
	// dc, err := dccl.Generate("tmp")
	// scheckErr(err)

	// images, err := cac.ImageStreams(proj.Name).List(metav1.ListOptions{})
	//checkErr(err)

	output := &app.ImageRef{
		Reference: imageapi.DockerImageReference{
			Registry: "docker.io",
			Name:     defaultImage,
		},
		AsImageStream: true,
	}
	// create our build based on source and input
	// TODO: we might need to pick a base image if this is STI
	// build := &BuildRef{Source: source, Output: output}
	// take the output image and wire it into a deployment config
	deploy := &app.DeploymentConfigRef{Images: []*app.ImageRef{output}}
	//outputRepo, _ := output.ImageStream()
	//buildConfig, _ := build.BuildConfig()
	deployConfig, _ := deploy.DeploymentConfig()
	deployConfig.Spec.Replicas = int32(1)
	deployConfig, err = dccl.Create(deployConfig)
	checkErr(err)

	// fmt.Printf("Using %#v scc...\n\n", provider.GetSCCName())
	fmt.Printf("\n")
	fmt.Printf("%#v\n\n", deployConfig)

	checkErr(os.RemoveAll(etcdt.DataDir))
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
