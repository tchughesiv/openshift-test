package main

import (
	"flag"
	"fmt"
	"math/rand"
	"os"
	"runtime"
	"time"

	"github.com/golang/glog"
	bp "github.com/openshift/origin/pkg/cmd/server/bootstrappolicy"
	"github.com/openshift/origin/pkg/cmd/server/kubernetes/node"
	nodeoptions "github.com/openshift/origin/pkg/cmd/server/kubernetes/node/options"
	cmdutil "github.com/openshift/origin/pkg/cmd/util"
	"github.com/openshift/origin/pkg/cmd/util/clientcmd"
	"github.com/openshift/origin/pkg/oc/cli"
	"github.com/openshift/origin/pkg/oc/cli/config"
	testutil "github.com/openshift/origin/test/util"
	testserver "github.com/openshift/origin/test/util/server"
	kclientcmd "k8s.io/client-go/tools/clientcmd"

	// install all APIs
	_ "github.com/openshift/origin/pkg/api/install"
	_ "k8s.io/kubernetes/pkg/api/install"
	_ "k8s.io/kubernetes/pkg/apis/autoscaling/install"
	_ "k8s.io/kubernetes/pkg/apis/batch/install"
	_ "k8s.io/kubernetes/pkg/apis/extensions/install"
)

// TODO - try moving away from running kubelet to, instead, direct container runtime interaction for compatibility on more OS's
// eventually offer in "oc"? initiate w/ --local flag?
var (
	n    = time.Now()
	args = os.Args
	d    = testutil.GetBaseDir() + "/log"
)

func init() {
	/*
		if runtime.GOOS == "darwin" {
			os.Setenv("TMPDIR", "/tmp")
		}
		// How can we further suppress the "startup" logs????
			logs.InitLogs()
			defer logs.FlushLogs()
			defer serviceability.BehaviorOnPanic(os.Getenv("OPENSHIFT_ON_PANIC"))()
			defer serviceability.Profile(os.Getenv("OPENSHIFT_PROFILE")).Stop()
	*/
	rand.Seed(time.Now().UTC().UnixNano())
	if len(os.Getenv("GOMAXPROCS")) == 0 {
		runtime.GOMAXPROCS(runtime.NumCPU())
	}

	v, b := os.LookupEnv("GLOG_V") // 0-10
	if b {
		var gv glog.Level
		checkErr(gv.Set(v))
	} else {
		gl := cmdutil.Env("GLOG_LEVEL", "ERROR") // INFO, ERROR, FATAL
		os.Args = []string{"glog", "-stderrthreshold=" + gl, "-logtostderr=false", "-log_dir=" + d}
		flag.Parse()
	}
}

func main() {
	os.Args = args
	var sccopts []string
	sflag := cmdutil.Env("OPENSHIFT_SCC", bp.SecurityContextConstraintRestricted)
	os.Setenv("TEST_ETCD_DIR", testutil.GetBaseDir()+"/etcd")

	if len(os.Args) == 1 {
		fmt.Printf("\nError: unknown command for %#v... must use \"run\"\n", os.Args[0])
		os.Exit(1)
	} else if os.Args[1] != "run" {
		fmt.Printf("\nError: unknown command %#v for %#v... must use \"run\"\n", os.Args[1], os.Args[0])
		os.Exit(1)
	}
	groups, users := bp.GetBoostrapSCCAccess(bp.DefaultOpenShiftInfraNamespace)
	for _, v := range bp.GetBootstrapSecurityContextConstraints(groups, users) {
		sccopts = append(sccopts, v.Name)
	}

	if !contains(sccopts, sflag) {
		fmt.Printf("\n%#v is not a valid scc. Must choose one of these:\n", sflag)
		for _, opt := range sccopts {
			fmt.Printf(" - %s\n", opt)
		}
		fmt.Printf("\n")
		os.Exit(1)
	}

	mconfig, nconfig, _, err := testserver.DefaultAllInOneOptions()
	checkErr(err)

	mkDir(d)
	mpath := testutil.GetBaseDir() + "/manifests"
	/*
		nconfig.PodManifestConfig = &configapi.PodManifestConfig{
			Path: mpath,
			FileCheckIntervalSeconds: int64(1),
		}
	*/
	kconfig, err := testserver.StartConfiguredMaster(mconfig)
	//kconfig, err := testserver.StartConfiguredAllInOne(mconfig, nconfig, components)
	checkErr(err)
	os.Setenv("KUBECONFIG", kconfig)
	//mkDir(mpath)

	s, err := nodeoptions.Build(*nconfig)
	checkErr(err)
	//_, err = node.New(*nconfig, s)
	nodeconfig, err := node.New(*nconfig, s)
	checkErr(err)

	cfg, err := config.NewOpenShiftClientConfigLoadingRules().Load()
	checkErr(err)
	defaultCfg := kclientcmd.NewDefaultClientConfig(*cfg, &kclientcmd.ConfigOverrides{})
	f := clientcmd.NewFactory(defaultCfg)
	namespace, _, err := f.DefaultNamespace()
	checkErr(err)
	kclient, err := f.ClientSet()
	checkErr(err)

	// wait for default serviceaccount to exist
	checkErr(testserver.WaitForServiceAccounts(kclient, namespace, []string{bp.DefaultServiceAccountName}))

	// modify scc settings before pod creation
	securityClient, err := f.OpenshiftInternalSecurityClient()
	checkErr(err)
	sccMod(sflag, namespace, securityClient)
	sccRm(sflag, namespace, securityClient)

	// execute cli command, force pod resource
	// kcommand := cli.CommandFor("kubectl")
	command := cli.CommandFor("oc")
	os.Args = append(os.Args, "--restart=Never")
	os.Args = append(os.Args, "--namespace="+namespace)
	if err := command.Execute(); err != nil {
		os.Exit(1)
	}

	// remove secrets from pod before kubelet runs
	recreatePod(kclient, namespace, mpath)

	fmt.Println("\nTotal start time:")
	fmt.Println(time.Since(n))

	runKubelet(nodeconfig)
	//checkErr(testserver.StartConfiguredNode(nconfig, components))
}
