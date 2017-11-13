package main

import (
	"fmt"
	"log"
	"math/rand"
	"os"
	"runtime"
	"testing"
	"time"

	"github.com/openshift/origin/pkg/bootstrap/docker/openshift"
	"github.com/openshift/origin/pkg/cmd/cli"
	bp "github.com/openshift/origin/pkg/cmd/server/bootstrappolicy"
	"github.com/openshift/origin/pkg/cmd/server/kubernetes/node"
	"github.com/openshift/origin/pkg/cmd/util/serviceability"
	testutil "github.com/openshift/origin/test/util"
	testserver "github.com/openshift/origin/test/util/server"
	"k8s.io/kubernetes/pkg/util/logs"

	// install all APIs # reference oc.go
	_ "github.com/openshift/origin/pkg/api/install"
	_ "k8s.io/kubernetes/pkg/api/install"
	_ "k8s.io/kubernetes/pkg/apis/autoscaling/install"
	_ "k8s.io/kubernetes/pkg/apis/batch/install"
	_ "k8s.io/kubernetes/pkg/apis/extensions/install"
)

// OPENSHIFT_SCC=anyuid ./sccoc new-app registry.centos.org/container-examples/starter-arbitrary-uid

func main() {
	var t *testing.T
	var sflag string
	var sccopts []string
	sccvar := "OPENSHIFT_SCC"
	defaultScc := bp.SecurityContextConstraintRestricted
	_, sccenv := os.LookupEnv(sccvar)

	if sccenv {
		sflag = os.Getenv(sccvar)
	} else {
		sflag = defaultScc
	}

	groups, users := bp.GetBoostrapSCCAccess(bp.DefaultOpenShiftInfraNamespace)
	bootstrappedConstraints := bp.GetBootstrapSecurityContextConstraints(groups, users)
	for _, v := range bootstrappedConstraints {
		sccopts = append(sccopts, v.Name)
		/*
			if v.Name == sflag {
				vtmp := v
				sccn = &vtmp
			}
		*/
	}

	if !contains(sccopts, sflag) {
		fmt.Printf("\n")
		fmt.Printf("%#v is not a valid scc. Must choose one of these:\n", sflag)
		for _, opt := range sccopts {
			fmt.Printf(" - %s\n", opt)
		}
		fmt.Printf("\n")
		os.Exit(1)
	}

	// How can supress the "startup" logs????
	etcdt := testutil.RequireEtcd(t)
	defer checkErr(os.RemoveAll(etcdt.DataDir))
	mconfig, nconfig, components, err := testserver.DefaultAllInOneOptions()
	checkErr(err)

	nodeconfig, err := node.BuildKubernetesNodeConfig(*nconfig, false, false)
	kserver := nodeconfig.KubeletServer
	// defer checkErr(dirCleanup(kserver.RootDirectory))
	kconfig, err := testserver.StartConfiguredAllInOne(mconfig, nconfig, components)

	// kubeCfg := kserver.KubeletConfiguration
	// kclient, err := testutil.GetClusterAdminKubeClient(kconfig)
	// checkErr(err)
	// oaclient, err := testutil.GetClusterAdminClient(kconfig)
	// checkErr(err)

	logs.InitLogs()
	defer logs.FlushLogs()
	defer serviceability.BehaviorOnPanic(os.Getenv("OPENSHIFT_ON_PANIC"))()
	defer serviceability.Profile(os.Getenv("OPENSHIFT_PROFILE")).Stop()

	rand.Seed(time.Now().UTC().UnixNano())
	if len(os.Getenv("GOMAXPROCS")) == 0 {
		runtime.GOMAXPROCS(runtime.NumCPU())
	}

	fmt.Printf("\n")
	os.Setenv("KUBECONFIG", kconfig)
	clArgs := os.Args
	command := cli.CommandFor("oc")

	fmt.Printf("\n")

	// modify scc settings accordingly
	if sflag != defaultScc {
		defaultsa := "system:serviceaccount:" + openshift.DefaultNamespace + ":" + bp.DefaultServiceAccountName
		os.Args = []string{"oc", "adm", "policy", "remove-scc-from-user", defaultScc, defaultsa}
		if err := command.Execute(); err != nil {
			os.Exit(1)
		}
		os.Args = []string{"oc", "adm", "policy", "add-scc-to-user", sflag, defaultsa}
		if err := command.Execute(); err != nil {
			os.Exit(1)
		}
		fmt.Printf("Changed scc for %#v...\n\n", defaultsa)
	}

	// openshift version
	os.Args = []string{"oc", "version"}
	if err := command.Execute(); err != nil {
		os.Exit(1)
	}

	// deploy registry
	fmt.Printf("\n")
	rmount := kserver.RootDirectory + "/registry"
	if _, err := os.Stat(rmount); os.IsNotExist(err) {
		os.Mkdir(rmount, 0750)
	}
	// os.Args = []string{"oc", "adm", "registry", "--service-account=registry", "--config=" + kconfig, "--mount-host=" + rmount}
	os.Args = []string{"oc", "adm", "registry"}
	if err := command.Execute(); err != nil {
		os.Exit(1)
	}

	fmt.Printf("\n")
	os.Args = []string{"oc", "get", "all", "--all-namespaces"}
	if err := command.Execute(); err != nil {
		os.Exit(1)
	}

	// ensure registry exists
	fmt.Printf("\n")
	os.Args = []string{"oc", "rollout", "status", "dc/docker-registry", "-w"}
	if err := command.Execute(); err != nil {
		os.Exit(1)
	}

	fmt.Printf("\n")
	os.Args = []string{"oc", "get", "all", "--all-namespaces"}
	if err := command.Execute(); err != nil {
		os.Exit(1)
	}

	fmt.Printf("Using %#v scc...\n\n", sflag)

	// execute cli command
	fmt.Printf("\n")
	os.Args = clArgs
	if err := command.Execute(); err != nil {
		os.Exit(1)
	}
}

func checkErr(err error) {
	if err != nil {
		log.Println(err)
	}
}

func contains(sccopts []string, sflag string) bool {
	for _, a := range sccopts {
		if a == sflag {
			return true
		}
	}
	return false
}

/*
func dirCleanup(rd string) error {
	searchDir := rd
	fileList := []string{}
	err := filepath.Walk(searchDir, func(path string, f os.FileInfo, err error) error {
		fileList = append(fileList, path)
		return nil
	})

	for _, file := range fileList {
		log.Println(file)
		if err := syscall.Unmount(
			file,
			syscall.MNT_DETACH,
		); err != nil {
			return err
		}

	}
	return err
}
*/
