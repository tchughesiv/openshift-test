package main

import (
	"fmt"
	"log"
	"math/rand"
	"os"
	"runtime"
	"strconv"
	"testing"
	"time"

	"github.com/openshift/origin/pkg/bootstrap/docker/openshift"
	"github.com/openshift/origin/pkg/cmd/admin/policy"
	"github.com/openshift/origin/pkg/cmd/cli"
	bp "github.com/openshift/origin/pkg/cmd/server/bootstrappolicy"
	"github.com/openshift/origin/pkg/cmd/util/serviceability"
	"github.com/openshift/origin/pkg/cmd/util/variable"
	securityapi "github.com/openshift/origin/pkg/security/apis/security"
	"github.com/openshift/origin/pkg/security/legacyclient"
	testutil "github.com/openshift/origin/test/util"
	testserver "github.com/openshift/origin/test/util/server"
	kapi "k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/util/logs"

	// install all APIs # reference oc.go
	_ "github.com/openshift/origin/pkg/api/install"
	_ "k8s.io/kubernetes/pkg/api/install"
	_ "k8s.io/kubernetes/pkg/apis/autoscaling/install"
	_ "k8s.io/kubernetes/pkg/apis/batch/install"
	_ "k8s.io/kubernetes/pkg/apis/extensions/install"
)

// CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags '-w' -o sccoc
// OPENSHIFT_SCC=anyuid ./sccoc new-app registry.centos.org/container-examples/starter-arbitrary-uid
// !!! WARNING: No Docker registry has been configured with the server. Automatic builds and deployments may not function.
// MUST create registry in default space...
// ??? ./origin/pkg/serviceaccounts/controllers/docker_registry_service.go

func main() {
	var t *testing.T
	var sflag string
	var sccopts []string
	var sccn *securityapi.SecurityContextConstraints
	sccvar := "OPENSHIFT_SCC"
	defaultScc := bp.SecurityContextConstraintRestricted
	_, sccenv := os.LookupEnv(sccvar)
	_ = sccn

	if sccenv {
		sflag = os.Getenv(sccvar)
	} else {
		sflag = defaultScc
	}

	groups, users := bp.GetBoostrapSCCAccess(bp.DefaultOpenShiftInfraNamespace)
	bootstrappedConstraints := bp.GetBootstrapSecurityContextConstraints(groups, users)
	for _, v := range bootstrappedConstraints {
		sccopts = append(sccopts, v.Name)
		if v.Name == sflag {
			vtmp := v
			sccn = &vtmp
		}
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
	etcdt, _ := testutil.RequireEtcd3(t)
	defer checkErr(os.RemoveAll(etcdt.DataDir))
	mconfig, nconfig, components, err := testserver.DefaultAllInOneOptions()
	checkErr(err)
	kconfig, err := testserver.StartConfiguredAllInOne(mconfig, nconfig, components)
	os.Setenv("KUBECONFIG", kconfig)
	kclient, err := testutil.GetClusterAdminKubeClient(kconfig)
	checkErr(err)
	// oaclient, err := testutil.GetClusterAdminClient(kconfig)
	// checkErr(err)
	// oaconfig, err := testutil.GetClusterAdminClientConfig(kconfig)
	// checkErr(err)

	// ./origin/pkg/cmd/admin/registry/registry.go
	// in, out, errout := os.Stdin, os.Stdout, os.Stderr
	// registry.NewCmdRegistry(f, fullName, "registry", out, errout),
	opts := &registry.RegistryOptions{
		Config: &registry.RegistryConfig{
			ImageTemplate:  variable.NewDefaultImageTemplate(),
			Name:           "registry",
			Labels:         registry.defaultLabel,
			Ports:          strconv.Itoa(registry.defaultPort),
			Volume:         "/registry",
			ServiceAccount: "registry",
			Replicas:       1,
			EnforceQuota:   false,
		},
	}
	//	kcmdutil.CheckErr(opts.Complete(f, cmd, out, errout, args))
	err = opts.RunCmdRegistry()
	if err == cmdutil.ErrExit {
		os.Exit(1)
	}
	//	kcmdutil.CheckErr(err)

	// modify scc settings accordingly
	if sflag != defaultScc {
		modifySCC := policy.SCCModificationOptions{
			SCCName:      defaultScc,
			SCCInterface: legacyclient.NewFromClient(kclient.Core().RESTClient()),
			Subjects: []kapi.ObjectReference{
				{
					Namespace: openshift.DefaultNamespace,
					Name:      bp.DefaultServiceAccountName,
					Kind:      "ServiceAccount",
				},
			},
		}
		err = modifySCC.RemoveSCC()
		checkErr(err)
		err = openshift.AddSCCToServiceAccount(kclient, sflag, bp.DefaultServiceAccountName, openshift.DefaultNamespace)
		checkErr(err)
	}

	logs.InitLogs()
	defer logs.FlushLogs()
	defer serviceability.BehaviorOnPanic(os.Getenv("OPENSHIFT_ON_PANIC"))()
	defer serviceability.Profile(os.Getenv("OPENSHIFT_PROFILE")).Stop()

	rand.Seed(time.Now().UTC().UnixNano())
	if len(os.Getenv("GOMAXPROCS")) == 0 {
		runtime.GOMAXPROCS(runtime.NumCPU())
	}

	fmt.Printf("\n")
	fmt.Printf("Using %#v scc...\n\n", sccn.Name)
	command := cli.CommandFor("sccoc")
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

func defaultPortForwarding() bool {
	// Defaults to true if running on Mac, with no DOCKER_HOST defined
	return runtime.GOOS == "darwin" && len(os.Getenv("DOCKER_HOST")) == 0
}
