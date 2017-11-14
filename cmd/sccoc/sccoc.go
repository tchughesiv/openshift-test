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
	configapi "github.com/openshift/origin/pkg/cmd/server/api"
	bp "github.com/openshift/origin/pkg/cmd/server/bootstrappolicy"
	"github.com/openshift/origin/pkg/cmd/server/etcd/etcdserver"
	"github.com/openshift/origin/pkg/cmd/util/serviceability"
	testutil "github.com/openshift/origin/test/util"
	testserver "github.com/openshift/origin/test/util/server"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/kubernetes/pkg/util/logs"

	// install all APIs # reference oc.go
	_ "github.com/openshift/origin/pkg/api/install"
	_ "k8s.io/kubernetes/pkg/api/install"
	_ "k8s.io/kubernetes/pkg/apis/autoscaling/install"
	_ "k8s.io/kubernetes/pkg/apis/batch/install"
	_ "k8s.io/kubernetes/pkg/apis/extensions/install"
)

// OPENSHIFT_SCC=anyuid ./sccoc new-app registry.centos.org/container-examples/starter-arbitrary-uid
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
	// switch to a more permanant etcd??? in tmp... then don't cleanup
	//if _, err := os.Stat(etcdData); os.IsNotExist(err) {
	//	err = os.Mkdir(etcdData, 0700)
	//	checkErr(err)
	//}
	//	_, out, errout := os.Stdin, os.Stdout, os.Stderr

	// _ = services.NewEtcd(etcdData)
	// checkErr(etcd.Start())

	// etcdt.Terminate(t)
	// os.RemoveAll(etcdt.DataDir)
	// s := etcdtest.NewUnsecuredEtcdTestClientServer(t)
	// url = etcdt.Client.Endpoints()[0]

	// defer os.RemoveAll(etcdData)
	// _, nconfig, kconfig, err := testserver.StartTestAllInOne()
	// checkErr(err)

	mconfig, err := RunEtcd()
	checkErr(err)
	_, nconfig, components, err := testserver.DefaultAllInOneOptions()
	checkErr(err)
	kconfig, err := testserver.StartConfiguredAllInOne(mconfig, nconfig, components)
	checkErr(err)
	oaclient, err := testutil.GetClusterAdminClient(kconfig)
	checkErr(err)

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
	defaultsa := "system:serviceaccount:" + openshift.DefaultNamespace + ":" + bp.DefaultServiceAccountName
	for _, a := range sccopts {
		if a == sflag {
			os.Args = []string{"oc", "adm", "policy", "add-scc-to-user", a, defaultsa}
			if err := command.Execute(); err != nil {
				os.Exit(1)
			}
			fmt.Printf("Added %#v scc to %#v...\n\n", a, defaultsa)
		} else {
			os.Args = []string{"oc", "adm", "policy", "remove-scc-from-user", a, defaultsa}
			if err := command.Execute(); err != nil {
				os.Exit(1)
			}
			// fmt.Printf("Removed %#v scc from %#v...\n\n", a, defaultsa)
		}
	}

	// openshift version
	os.Args = []string{"oc", "version"}
	if err := command.Execute(); err != nil {
		os.Exit(1)
	}

	fmt.Printf("\n")

	// deploy registry
	rmount := nconfig.VolumeDirectory + "/registry"
	dc := oaclient.DeploymentConfigs(openshift.DefaultNamespace)
	dcg, err := dc.Get("docker-registry", metav1.GetOptions{})
	checkErr(err)
	if dcg.GetName() != "" {
		if dcg.Status.Conditions[0].Status != "True" {
			// ensure registry comes up
			fmt.Printf("\n")
			os.Args = []string{"oc", "rollout", "latest", "dc/docker-registry"}
			if err := command.Execute(); err != nil {
				os.Exit(1)
			}
			fmt.Printf("\n")
			os.Args = []string{"oc", "rollout", "status", "dc/docker-registry", "-w"}
			if err := command.Execute(); err != nil {
				os.Exit(1)
			}
		}
	}
	dcg, err = dc.Get("docker-registry", metav1.GetOptions{})
	checkErr(err)
	if dcg.GetName() == "" {
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

	fmt.Printf("\n")
	os.Args = []string{"oc", "get", "all", "--all-namespaces"}
	if err := command.Execute(); err != nil {
		os.Exit(1)
	}

	fmt.Printf("\n")
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

// RunEtcd inits etcd
func RunEtcd() (*configapi.MasterConfig, error) {
	var t *testing.T

	masterConfig, err := testserver.DefaultMasterOptionsWithTweaks(true /*start etcd server*/, false /*don't use default ports*/)
	if err != nil {
		return nil, err
	}

	etcdConfig := masterConfig.EtcdConfig
	masterConfig.EtcdConfig = nil
	masterConfig.DNSConfig = nil

	etcdserver.RunEtcd(etcdConfig)
	etcdt := testutil.RequireEtcd(t)
	etcdt.Terminate(t)
	// checkErr(os.RemoveAll(etcdt.DataDir))

	return masterConfig, err
}
