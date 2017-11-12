package main

import (
	"fmt"
	"log"
	"os"
	"testing"

	"github.com/spf13/cobra"

	"github.com/openshift/origin/pkg/bootstrap/docker/openshift"
	"github.com/openshift/origin/pkg/cmd/cli"
	bp "github.com/openshift/origin/pkg/cmd/server/bootstrappolicy"
	projectapi "github.com/openshift/origin/pkg/project/apis/project"
	allocator "github.com/openshift/origin/pkg/security"
	admtesting "github.com/openshift/origin/pkg/security/admission/testing"
	securityapi "github.com/openshift/origin/pkg/security/apis/security"
	testutil "github.com/openshift/origin/test/util"
	testserver "github.com/openshift/origin/test/util/server"
)

// command options/description reference ???
// https://github.com/openshift/origin/blob/release-3.6/pkg/cmd/cli/cli.go
// CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags '-w' -o sccoc

// sccoc --scc=anyuid new-app alpine:latest

func main() {
	defaultScc := "restricted"
	//defaultImage := "docker.io/centos:latest"
	var t *testing.T
	var sccopts []string
	var sccn *securityapi.SecurityContextConstraints
	_ = sccn

	if len(os.Args) > 1 {
		defaultScc = os.Args[len(os.Args)-1]
	}

	groups, users := bp.GetBoostrapSCCAccess(bp.DefaultOpenShiftInfraNamespace)
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

	kconfig, err := testserver.StartConfiguredAllInOne(mconfig, nconfig, components)
	cac, err := testutil.GetClusterAdminClient(kconfig)
	checkErr(err)
	kclient, err := testutil.GetClusterAdminKubeClient(kconfig)
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

	ns := admtesting.CreateNamespaceForTest()
	ns.Name = testutil.RandomNamespace("tmp")
	ns.Annotations[allocator.UIDRangeAnnotation] = "1000100000/10000"
	ns.Annotations[allocator.MCSAnnotation] = "s9:z0,z1"
	ns.Annotations[allocator.SupplementalGroupsAnnotation] = "1000100000/10000"

	project := &projectapi.Project{}
	project.Name = ns.Name
	project.Annotations = ns.Annotations

	projcl := cac.Projects()
	proj, err := projcl.Create(project)
	checkErr(err)

	// modify scc settings accordingly
	if defaultScc != "restricted" {
		err = openshift.AddSCCToServiceAccount(kclient, defaultScc, bp.DefaultServiceAccountName, proj.Name)
		checkErr(err)
	}

	// use new-app cmd to deploy specified image
	var cmd *cobra.Command
	in, out, errout := os.Stdin, os.Stdout, os.Stderr

	cmd = cli.NewCommandCLI("sccoc", "sccoc", in, out, errout)
	test := cmd.Commands()
	fmt.Printf("\n")
	fmt.Printf("%#v\n\n", test)

	/*
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
		// outputRepo, _ := output.ImageStream()
		// buildConfig, _ := build.BuildConfig()
		// take the output image and wire it into a deployment config
		deploy := &app.DeploymentConfigRef{Images: []*app.ImageRef{output}}
		deployConfig, _ := deploy.DeploymentConfig()
		deployConfig.Spec.Replicas = int32(1)
		deployConfig, err = dccl.Create(deployConfig)
		checkErr(err)
	*/
	// fmt.Printf("Using %#v scc...\n\n", provider.GetSCCName())

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
