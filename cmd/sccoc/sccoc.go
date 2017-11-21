package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"runtime"
	"time"

	"github.com/ghodss/yaml"
	configapi "github.com/openshift/origin/pkg/cmd/server/api"
	bp "github.com/openshift/origin/pkg/cmd/server/bootstrappolicy"
	"github.com/openshift/origin/pkg/cmd/server/kubernetes/node"
	nodeoptions "github.com/openshift/origin/pkg/cmd/server/kubernetes/node/options"
	cmdutil "github.com/openshift/origin/pkg/cmd/util"
	"github.com/openshift/origin/pkg/cmd/util/clientcmd"
	"github.com/openshift/origin/pkg/cmd/util/serviceability"
	"github.com/openshift/origin/pkg/oc/cli"
	"github.com/openshift/origin/pkg/oc/cli/config"
	testutil "github.com/openshift/origin/test/util"
	testserver "github.com/openshift/origin/test/util/server"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kclientcmd "k8s.io/client-go/tools/clientcmd"
	"k8s.io/kubernetes/cmd/kubelet/app"
	"k8s.io/kubernetes/pkg/util/logs"

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

// Flow -
// 1. start test master
// 2. execute "run" against to generate pod yaml w/ scc
// 3. export yaml w/ sc settings to pod manifest dir
// 4. start real kubelet pointed to manifest dir - should deploy pod

func main() {
	t := time.Now()
	var sccopts []string
	sflag := cmdutil.Env("OPENSHIFT_SCC", bp.SecurityContextConstraintRestricted)
	os.Setenv("TEST_ETCD_DIR", testutil.GetBaseDir()+"/etcd")

	if os.Args[1] != "run" {
		fmt.Printf("\nError: unknown command %#v for %#v... must use \"run\"\n", os.Args[1], os.Args[0])
		os.Exit(1)
	}
	clArgs := os.Args
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

	// How can supress the "startup" logs????
	//os.Setenv("KUBELET_NETWORK_ARGS", "")
	mconfig, nconfig, _, err := testserver.DefaultAllInOneOptions()
	checkErr(err)
	kconfig := testutil.KubeConfigPath()
	os.Setenv("KUBECONFIG", kconfig)
	mpath := testutil.GetBaseDir() + "/manifests"
	nconfig.PodManifestConfig = &configapi.PodManifestConfig{
		Path: mpath,
		FileCheckIntervalSeconds: int64(5),
	}
	_, err = testserver.StartConfiguredMaster(mconfig)
	checkErr(err)
	t2 := time.Since(t)

	if _, err := os.Stat(mpath); os.IsNotExist(err) {
		os.Mkdir(mpath, 0755)
	}
	s, err := nodeoptions.Build(*nconfig)
	checkErr(err)
	nodeconfig, err := node.New(*nconfig, s)
	checkErr(err)
	kubeDeps := nodeconfig.KubeletDeps

	cfg, err := config.NewOpenShiftClientConfigLoadingRules().Load()
	checkErr(err)
	defaultCfg := kclientcmd.NewDefaultClientConfig(*cfg, &kclientcmd.ConfigOverrides{})
	f := clientcmd.NewFactory(defaultCfg)
	namespace, _, err := f.DefaultNamespace()
	checkErr(err)
	kclient, err := f.ClientSet()
	checkErr(err)

	// wait for default serviceaccount to exist
	_, err = kclient.Core().ServiceAccounts(namespace).Get(bp.DefaultServiceAccountName, metav1.GetOptions{})
	i := 0
	for err != nil {
		//fmt.Printf("\n%#v\n", i)
		if i < 80 {
			time.Sleep(time.Millisecond * 250)
			_, err = kclient.Core().ServiceAccounts(namespace).Get(bp.DefaultServiceAccountName, metav1.GetOptions{})
		}
		i++
	}

	logs.InitLogs()
	defer logs.FlushLogs()
	defer serviceability.BehaviorOnPanic(os.Getenv("OPENSHIFT_ON_PANIC"))()
	defer serviceability.Profile(os.Getenv("OPENSHIFT_PROFILE")).Stop()

	rand.Seed(time.Now().UTC().UnixNano())
	if len(os.Getenv("GOMAXPROCS")) == 0 {
		runtime.GOMAXPROCS(runtime.NumCPU())
	}
	//in, out, errout := os.Stdin, os.Stdout, os.Stderr
	command := cli.CommandFor("oc")
	// kcommand := cli.CommandFor("kubectl")

	// modify scc settings accordingly
	securityClient, err := f.OpenshiftInternalSecurityClient()
	checkErr(err)
	sccmod(sflag, namespace, securityClient)
	sccrm(sflag, namespace, securityClient)
	t3 := time.Since(t)

	// execute cli command
	clArgs = append(clArgs, "--restart=Never")
	clArgs = append(clArgs, "--namespace="+namespace)
	os.Args = clArgs
	if err := command.Execute(); err != nil {
		os.Exit(1)
	}

	fmt.Printf("\n")
	podint := kclient.Core().Pods(namespace)
	podl, err := podint.List(metav1.ListOptions{})
	checkErr(err)
	pod, err := podint.Get(podl.Items[0].GetName(), metav1.GetOptions{})
	checkErr(err)

	// mirror pod mods
	//pod.Kind = "Pod"
	//pod.APIVersion = "v1"
	pod.Spec.ServiceAccountName = ""
	pod.ObjectMeta.ResourceVersion = ""

	podyf := mpath + "/" + pod.Name + "-pod.yaml"
	jpod, err := json.Marshal(pod)
	checkErr(err)
	pyaml, err := yaml.JSONToYAML(jpod)
	checkErr(err)
	ioutil.WriteFile(podyf, pyaml, os.FileMode(0600))

	// Run kubelet
	// requires higher max user watches for file method...
	// sudo sysctl fs.inotify.max_user_watches=524288
	// ?? make the change permanent, edit the file /etc/sysctl.conf and add the line to the end of the file
	// remove serviceaccount, secrets, resourceVersion from pod yaml before processing as mirror pod
	s.RunOnce = true
	err = app.Run(s, kubeDeps)
	checkErr(err)

	fmt.Println(string(jpod))

	fmt.Println("time from post master, post scc, to end")
	fmt.Println(t2)
	fmt.Println(t3)
	fmt.Println(time.Since(t))

	/*
		os.Args = []string{"oc", "get", "pod", pod.GetName(), "--namespace=" + namespace, "--output=yaml"}
		if err := kcommand.Execute(); err != nil {
			os.Exit(1)
		}
	*/
}
