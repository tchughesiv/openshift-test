package main

import (
	"fmt"
	"log"
	"math/rand"
	"os"
	"runtime"
	"time"

	configapi "github.com/openshift/origin/pkg/cmd/server/api"
	bp "github.com/openshift/origin/pkg/cmd/server/bootstrappolicy"
	"github.com/openshift/origin/pkg/cmd/server/kubernetes/node"
	nodeoptions "github.com/openshift/origin/pkg/cmd/server/kubernetes/node/options"
	"github.com/openshift/origin/pkg/cmd/util/serviceability"
	"github.com/openshift/origin/pkg/oc/cli"
	testserver "github.com/openshift/origin/test/util/server"
	"k8s.io/kubernetes/pkg/util/logs"

	// install all APIs
	_ "github.com/openshift/origin/pkg/api/install"
	_ "k8s.io/kubernetes/pkg/api/install"
	_ "k8s.io/kubernetes/pkg/apis/autoscaling/install"
	_ "k8s.io/kubernetes/pkg/apis/batch/install"
	_ "k8s.io/kubernetes/pkg/apis/extensions/install"
)

// OPENSHIFT_SCC=anyuid ./sccoc run test --image=registry.centos.org/container-examples/starter-arbitrary-uid
// ./origin/cmd/oc/oc.go
// maybe limit to just new-app & run???

func main() {
	var sflag string
	var sccopts []string
	sccvar := "OPENSHIFT_SCC"
	os.Setenv("TEST_ETCD_DIR", "/tmp/etcdtest")
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
	mconfig, nconfig, components, err := testserver.DefaultAllInOneOptions()
	checkErr(err)
	mpath := nconfig.VolumeDirectory + "/manifests"
	if _, err := os.Stat(mpath); os.IsNotExist(err) {
		os.Mkdir(mpath, 0750)
	}
	nconfig.PodManifestConfig = &configapi.PodManifestConfig{
		Path: mpath,
		FileCheckIntervalSeconds: int64(5),
	}
	kconfig, err := testserver.StartConfiguredAllInOne(mconfig, nconfig, components)
	checkErr(err)
	//oaclient, err := testutil.GetClusterAdminClient(kconfig)
	//checkErr(err)

	os.Setenv("KUBECONFIG", kconfig)
	clArgs := os.Args
	logs.InitLogs()
	defer logs.FlushLogs()
	defer serviceability.BehaviorOnPanic(os.Getenv("OPENSHIFT_ON_PANIC"))()
	defer serviceability.Profile(os.Getenv("OPENSHIFT_PROFILE")).Stop()

	rand.Seed(time.Now().UTC().UnixNano())
	if len(os.Getenv("GOMAXPROCS")) == 0 {
		runtime.GOMAXPROCS(runtime.NumCPU())
	}
	command := cli.CommandFor("oc")
	kcommand := cli.CommandFor("kubectl")

	fmt.Printf("\n")

	// modify scc settings accordingly
	defaultsa := "system:serviceaccount:default:" + bp.DefaultServiceAccountName
	for _, a := range sccopts {
		if a == sflag {
			os.Args = []string{"oc", "adm", "policy", "add-scc-to-user", a, defaultsa}
			if err := command.Execute(); err != nil {
				os.Exit(1)
			}
		} else {
			os.Args = []string{"oc", "adm", "policy", "remove-scc-from-user", a, defaultsa}
			if err := command.Execute(); err != nil {
				os.Exit(1)
			}
		}
	}

	fmt.Printf("\n")
	fmt.Printf("\n")
	os.Args = []string{"oc", "get", "all", "--all-namespaces"}
	if err := command.Execute(); err != nil {
		os.Exit(1)
	}

	// Run kubelet
	s, err := nodeoptions.Build(*nconfig)
	checkErr(err)
	nodeconfig, err := node.New(*nconfig, s)
	checkErr(err)
	fmt.Printf("%#v\n", nodeconfig.DockerClient)
	// kserver = nodeconfig.KubeletServer
	// kserver.ContainerRuntime = "docker"
	nodeconfig.RunKubelet()

	// execute cli command
	fmt.Printf("\n")
	os.Args = clArgs
	if err := kcommand.Execute(); err != nil {
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
	// deploy registry
	rmount := nconfig.VolumeDirectory + "/registry"
	dc := oaclient.DeploymentConfigs(openshift.DefaultNamespace)
	dcg, err := dc.Get("docker-registry", metav1.GetOptions{})
	checkErr(err)
	if dcg.GetName() != "" {
		if dcg.Status.ReadyReplicas == 0 {
			fmt.Printf("\n")
			os.Args = []string{"oc", "delete", "dc/docker-registry", "svc/docker-registry"}
			if err := command.Execute(); err != nil {
				os.Exit(1)
			}
			os.Args = []string{"oc", "delete", "clusterrolebinding.authorization.openshift.io", "registry-registry-role"}
			if err := command.Execute(); err != nil {
				os.Exit(1)
			}
			os.Args = []string{"oc", "delete", "sa", "registry"}
			if err := command.Execute(); err != nil {
				os.Exit(1)
			}
			// ?? add a loop until dc cleared???
		}
	}
	dcg, err = dc.Get("docker-registry", metav1.GetOptions{})
	checkErr(err)
	if dcg.GetName() == "" {
		fmt.Printf("\n")
		if _, err := os.Stat(rmount); os.IsNotExist(err) {
			os.Mkdir(rmount, 0750)
		}
		os.Args = []string{"oc", "adm", "registry", "--service-account=registry", "--config=" + kconfig, "--mount-host=" + rmount}
		// os.Args = []string{"oc", "adm", "registry"}
		if err := command.Execute(); err != nil {
			os.Exit(1)
		}

		// ensure registry comes up
		fmt.Printf("\n")
		os.Args = []string{"oc", "rollout", "status", "dc/docker-registry", "-w"}
		if err := command.Execute(); err != nil {
			os.Exit(1)
		}
	}
*/
