package main

import (
	"fmt"
	"log"
	"os"

	bp "github.com/openshift/origin/pkg/cmd/server/bootstrappolicy"
	"github.com/openshift/origin/pkg/diagnostics/network"
	allocator "github.com/openshift/origin/pkg/security"
	"github.com/openshift/origin/pkg/security/admission/testing"
	securityapi "github.com/openshift/origin/pkg/security/apis/security"
	"github.com/openshift/origin/pkg/security/scc"
	testutil "github.com/openshift/origin/test/util"
	testserver "github.com/openshift/origin/test/util/server"
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
	var sccopts []string
	var sccn *securityapi.SecurityContextConstraints
	if len(os.Args) > 1 {
		defaultScc = os.Args[len(os.Args)-1]
	}

	// nsa := testing.CreateSAForTest()
	ns := testing.CreateNamespaceForTest()
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
		fmt.Printf("%#v is not a valid scc. Must choose one of these:\n", defaultScc)
		fmt.Printf("%v\n", sccopts)
		os.Exit(1)
	}

	_, err := testserver.DefaultMasterOptionsWithTweaks(true, false)
	checkErr(err)
	kconfig := testutil.KubeConfigPath()
	clusterAdminKubeClientset, err := testutil.GetClusterAdminKubeClient(kconfig)
	checkErr(err)

	/*
		clusterAdminClientConfig, err := testutil.GetClusterAdminClientConfig(kconfig)
		checkErr(err)
		clusterAdminClient, err := testutil.GetClusterAdminClient(kconfig)
		checkErr(err)
	*/

	fmt.Printf("\n")
	sccp, ns, err := scc.CreateProviderFromConstraint(ns.Name, ns, sccn, clusterAdminKubeClientset)
	checkErr(err)

	//  testis := testgen.MockImageStream("centos", "docker.io/centos", map[string]string{"latest": "latest"})
	//	fmt.Printf("%#v\n\n", testis)

	// testpod := testutil.CreatePodFromImage(testis, "latest", ns.Name)
	testpod := network.GetTestPod("docker.io/centos:latest", "tcp", "tmp", "localhost", 12000)

	/*
		psc, _, err := sccp.CreatePodSecurityContext(testpod)
		_ = psc
		checkErr(err)
	*/

	testcontainer := testpod.Spec.Containers[0]
	tc := &testcontainer
	sc, err := sccp.CreateContainerSecurityContext(testpod, tc)
	checkErr(err)

	// fmt.Printf("%#v\n\n", psc)
	// fmt.Printf("%#v\n\n", testcontainer)
	fmt.Printf("\n%#v\n\n", sc)
	fmt.Printf("%#v\n\n", sc.Capabilities)
	fmt.Printf("%#v\n\n", sc.SELinuxOptions)
	// fmt.Printf("%#v\n\n", sc.RunAsUser)
	fmt.Printf("Using %#v scc...\n\n", sccp.GetSCCName())

	// !!!  convert specified scc definition into container runtime configs - using origin code??? - search for cap to docker conversion code
	// !!!  run image accordingly directly against container runtime... no ocp/k8s involvement

	// ?? reference for container runtime -
	// vendor/github.com/openshift/origin/vendor/k8s.io/kubernetes/pkg/kubelet/kubelet.go
	// vendor/github.com/openshift/origin/vendor/k8s.io/kubernetes/pkg/kubectl/run_test.go

	// kubectl run reference: https://github.com/openshift/kubernetes/blob/openshift-1.6-20170501/pkg/kubectl/run_test.go
}
