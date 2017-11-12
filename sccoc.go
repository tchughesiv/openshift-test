package main

import (
	"flag"
	"fmt"
	"log"
	"math/rand"
	"os"
	"runtime"
	"testing"
	"time"

	"github.com/openshift/origin/pkg/bootstrap/docker/openshift"
	"github.com/openshift/origin/pkg/cmd/admin/policy"
	"github.com/openshift/origin/pkg/cmd/cli"
	bp "github.com/openshift/origin/pkg/cmd/server/bootstrappolicy"
	"github.com/openshift/origin/pkg/cmd/util/serviceability"
	securityapi "github.com/openshift/origin/pkg/security/apis/security"
	"github.com/openshift/origin/pkg/security/legacyclient"
	testutil "github.com/openshift/origin/test/util"
	testserver "github.com/openshift/origin/test/util/server"
	kapi "k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/util/logs"

	// install all APIs
	_ "github.com/openshift/origin/pkg/api/install"
	_ "k8s.io/kubernetes/pkg/api/install"
	_ "k8s.io/kubernetes/pkg/apis/autoscaling/install"
	_ "k8s.io/kubernetes/pkg/apis/batch/install"
	_ "k8s.io/kubernetes/pkg/apis/extensions/install"
)

// CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags '-w' -o sccoc
// sccoc --scc=anyuid new-app alpine:latest

func main() {
	var strFlag = flag.String("scc", "restricted", "Description")
	flag.Parse()
	println(*strFlag)

	defaultScc := "restricted"
	sflag := *strFlag
	var t *testing.T
	var sccopts []string
	var sccn *securityapi.SecurityContextConstraints
	_ = sccn

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
	etcdt := testutil.RequireEtcd(t)
	defer checkErr(os.RemoveAll(etcdt.DataDir))

	mconfig, nconfig, components, err := testserver.DefaultAllInOneOptions()
	checkErr(err)

	kconfig, err := testserver.StartConfiguredAllInOne(mconfig, nconfig, components)
	kclient, err := testutil.GetClusterAdminKubeClient(kconfig)
	checkErr(err)
	//cac, err := testutil.GetClusterAdminClient(kconfig)
	//checkErr(err)

	// modify scc settings accordingly
	if sflag != defaultScc {
		modifySCC := policy.SCCModificationOptions{
			SCCName:      defaultScc,
			SCCInterface: legacyclient.NewFromClient(kclient.Core().RESTClient()),
			Subjects: []kapi.ObjectReference{
				{
					Namespace: bp.DefaultOpenShiftInfraNamespace,
					Name:      bp.DefaultServiceAccountName,
					Kind:      "ServiceAccount",
				},
			},
		}
		err = modifySCC.RemoveSCC()
		checkErr(err)
		err = openshift.AddSCCToServiceAccount(kclient, sflag, bp.DefaultServiceAccountName, bp.DefaultOpenShiftInfraNamespace)
		checkErr(err)
	}

	// fmt.Printf("\n")
	// fmt.Printf("%#v\n\n", proj)

	// !!! reference https://github.com/openshift/origin/blob/release-3.6/cmd/oc/oc.go
	logs.InitLogs()
	defer logs.FlushLogs()
	defer serviceability.BehaviorOnPanic(os.Getenv("OPENSHIFT_ON_PANIC"))()
	defer serviceability.Profile(os.Getenv("OPENSHIFT_PROFILE")).Stop()

	rand.Seed(time.Now().UTC().UnixNano())
	if len(os.Getenv("GOMAXPROCS")) == 0 {
		runtime.GOMAXPROCS(runtime.NumCPU())
	}

	// !! force connection to test server client instead
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
