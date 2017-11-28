package main

import (
	"flag"
	"fmt"
	"math/rand"
	"os"
	"runtime"
	"time"

	"github.com/golang/glog"
	configapi "github.com/openshift/origin/pkg/cmd/server/api"
	bp "github.com/openshift/origin/pkg/cmd/server/bootstrappolicy"
	"github.com/openshift/origin/pkg/cmd/server/kubernetes/node"
	nodeoptions "github.com/openshift/origin/pkg/cmd/server/kubernetes/node/options"
	cmdutil "github.com/openshift/origin/pkg/cmd/util"
	"github.com/openshift/origin/pkg/cmd/util/clientcmd"
	"github.com/openshift/origin/pkg/oc/cli"
	"github.com/openshift/origin/pkg/oc/cli/config"
	testutil "github.com/openshift/origin/test/util"
	testserver "github.com/openshift/origin/test/util/server"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kclientcmd "k8s.io/client-go/tools/clientcmd"

	// install all APIs
	_ "github.com/openshift/origin/pkg/api/install"
	_ "k8s.io/kubernetes/pkg/api/install"
	_ "k8s.io/kubernetes/pkg/apis/autoscaling/install"
	_ "k8s.io/kubernetes/pkg/apis/batch/install"
	_ "k8s.io/kubernetes/pkg/apis/extensions/install"
)

// OPENSHIFT_SCC=nonroot origin/_output/local/bin/linux/amd64/sccoc run testpod --image=registry.centos.org/container-examples/starter-arbitrary-uid
// OPENSHIFT_SCC=nonroot sccoc run testpod --image=registry.centos.org/container-examples/starter-arbitrary-uid
// ./origin/cmd/oc/oc.go
// sudo KUBECONFIG=/tmp/openshift-integration/openshift.local.config/master/admin.kubeconfig oc get all --all-namespaces
// maybe limit to just run???

// 1. start test master
// 2. execute "run" against to generate pod yaml w/ scc
// 3. export yaml w/ sc settings to pod manifest dir
// 4. start kubelet pointed to manifest dir - should deploy pod
var (
	n    = time.Now()
	args = os.Args
	d    = testutil.GetBaseDir() + "/log"
)

func init() {
	/*
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

	// How can suppress the "startup" logs????
	//os.Setenv("KUBELET_NETWORK_ARGS", "")
	mconfig, nconfig, components, err := testserver.DefaultAllInOneOptions()
	checkErr(err)

	mkDir(d)
	mpath := testutil.GetBaseDir() + "/manifests"
	nconfig.PodManifestConfig = &configapi.PodManifestConfig{
		Path: mpath,
		FileCheckIntervalSeconds: int64(1),
	}

	//kconfig, err := testserver.StartConfiguredMaster(mconfig)
	kconfig, err := testserver.StartConfiguredAllInOne(mconfig, nconfig, components)
	checkErr(err)
	os.Setenv("KUBECONFIG", kconfig)
	mkDir(mpath)
	s, err := nodeoptions.Build(*nconfig)
	checkErr(err)
	_, err = node.New(*nconfig, s)
	//nodeconfig, err := node.New(*nconfig, s)
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
	n2 := time.Since(n)

	// modify scc settings accordingly
	securityClient, err := f.OpenshiftInternalSecurityClient()
	checkErr(err)
	sccMod(sflag, namespace, securityClient)
	sccRm(sflag, namespace, securityClient)
	si := kclient.Core().Secrets(namespace)
	se, err := si.List(metav1.ListOptions{})
	checkErr(err)
	for _, z := range se.Items {
		checkErr(si.Delete(z.Name, metav1.DeleteOptions{}),
	}

	// execute cli command
	// kcommand := cli.CommandFor("kubectl")
	command := cli.CommandFor("oc")
	os.Args = append(os.Args, "--restart=Never")
	os.Args = append(os.Args, "--namespace="+namespace)
	if err := command.Execute(); err != nil {
		os.Exit(1)
	}

	// remove token from default sa before pod creation
	exportPod(kclient, namespace, mpath)
	//runKubelet(nodeconfig)

	fmt.Println("\ntime until master ready...")
	fmt.Println(n2)
	fmt.Println("\nTotal time.")
	fmt.Println(time.Since(n))
	fmt.Println(se.Items)

	/*
		os.Args = []string{"oc", "get", "pod", pod.GetName(), "--namespace=" + namespace, "--output=yaml"}
		if err := kcommand.Execute(); err != nil {
			os.Exit(1)
		}
	*/
}
