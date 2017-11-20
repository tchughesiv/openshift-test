package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"os"
	"runtime"
	"time"

	configapi "github.com/openshift/origin/pkg/cmd/server/api"
	"github.com/openshift/origin/pkg/cmd/server/api/latest"
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

// OPENSHIFT_SCC=anyuid ./sccoc run test --image=registry.centos.org/container-examples/starter-arbitrary-uid
// ./origin/cmd/oc/oc.go
// sudo KUBECONFIG=/tmp/openshift-integration/openshift.local.config/master/admin.kubeconfig oc get all --all-namespaces
// maybe limit to just run???

// Flow -
// 1. start test master
// 2. execute "run" against to generate pod yaml w/ scc
// 3. export yaml w/ sc settings to pod manifest dir
// 4. start real kubelet pointed to manifest dir - should deploy pod

func main() {
	var sccopts []string
	namespace := "tmp"
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
	// os.Setenv("KUBELET_NETWORK_ARGS", "")
	mconfig, nconfig, _, err := testserver.DefaultAllInOneOptions()
	checkErr(err)
	mpath := testutil.GetBaseDir() + "/manifests"
	nconfig.PodManifestConfig = &configapi.PodManifestConfig{
		Path: mpath,
		FileCheckIntervalSeconds: int64(5),
	}
	// kconfig, err := testserver.StartConfiguredAllInOne(mconfig, nconfig, components)
	kconfig, err := testserver.StartConfiguredMaster(mconfig)
	os.Setenv("KUBECONFIG", kconfig)
	if _, err := os.Stat(mpath); os.IsNotExist(err) {
		os.Mkdir(mpath, 0755)
	}
	s, err := nodeoptions.Build(*nconfig)
	checkErr(err)
	nodeconfig, err := node.New(*nconfig, s)
	checkErr(err)
	kubeDeps := nodeconfig.KubeletDeps

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

	/*
		os.Args = []string{"oc", "new-project", namespace}
		if err := command.Execute(); err != nil {
			os.Exit(1)
		}
	*/
	// Run kubelet
	// requires higher max user watches for file method...
	// sudo sysctl fs.inotify.max_user_watches=524288
	// ?? make the change permanent, edit the file /etc/sysctl.conf and add the line to the end of the file

	cfg, err := config.NewOpenShiftClientConfigLoadingRules().Load()
	checkErr(err)
	defaultCfg := kclientcmd.NewDefaultClientConfig(*cfg, &kclientcmd.ConfigOverrides{})
	f := clientcmd.NewFactory(defaultCfg)
	kclient, err := f.ClientSet()
	checkErr(err)
	_, err = f.OpenshiftInternalAppsClient()
	checkErr(err)

	clusterAdminClientConfig, err := testutil.GetClusterAdminClientConfig(kconfig)
	checkErr(err)
	_, _, err = testserver.CreateNewProject(clusterAdminClientConfig, namespace, "tommy")
	checkErr(err)

	_, err = kclient.Core().ServiceAccounts(namespace).Get(bp.DefaultServiceAccountName, metav1.GetOptions{})
	i := 0
	for err != nil {
		//fmt.Printf("\n%#v\n", i)
		if i < 5 {
			time.Sleep(time.Second * 3)
			_, err = kclient.Core().ServiceAccounts(namespace).Get(bp.DefaultServiceAccountName, metav1.GetOptions{})
		}
		i++
	}

	/*
		sas, err := kclient.Core().ServiceAccounts(namespace).List(metav1.ListOptions{})
		checkErr(err)
		fmt.Println(sas.Items)
	*/

	// modify scc settings accordingly
	sa := "system:serviceaccount:" + namespace + ":" + bp.DefaultServiceAccountName
	patch, err := json.Marshal(scc{Priority: 1})
	checkErr(err)

	os.Args = []string{"oc", "patch", "scc", sflag, "--patch", string(patch)}
	if err := command.Execute(); err != nil {
		os.Exit(1)
	}

	if sflag != bp.SecurityContextConstraintRestricted {
		os.Args = []string{"oc", "adm", "policy", "add-scc-to-user", sflag, sa}
		if err := command.Execute(); err != nil {
			os.Exit(1)
		}
		os.Args = []string{"oc", "adm", "policy", "remove-scc-from-user", bp.SecurityContextConstraintRestricted, sa}
		if err := command.Execute(); err != nil {
			os.Exit(1)
		}
	}

	if sflag != bp.SecurityContextConstraintsAnyUID {
		os.Args = []string{"oc", "adm", "policy", "remove-scc-from-group", bp.SecurityContextConstraintsAnyUID, "system:cluster-admins"}
		if err := command.Execute(); err != nil {
			os.Exit(1)
		}
	}

	/*
		for _, a := range sccopts {
			if a == sflag {
				os.Args = []string{"oc", "adm", "policy", "add-scc-to-user", a, sa}
				if err := command.Execute(); err != nil {
					os.Exit(1)
				}
			} else {
				os.Args = []string{"oc", "adm", "policy", "remove-scc-from-user", a, sa}
				if err := command.Execute(); err != nil {
					os.Exit(1)
				}
			}
		}
	*/

	ns, err := kclient.Core().Namespaces().Get(namespace, metav1.GetOptions{})
	checkErr(err)
	fmt.Println(ns.Annotations)

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

	podyf := mpath + "/" + pod.Name + ".yaml"
	//pyaml, err := yaml.JSONToYAML(pod.GetObjectMeta())
	pyaml, err := latest.WriteYAML(pod)
	checkErr(err)
	ioutil.WriteFile(podyf, pyaml, os.FileMode(0600))

	s.RunOnce = true
	err = app.Run(s, kubeDeps)
	checkErr(err)

	os.Args = []string{"oc", "get", "pod", pod.GetName(), "--namespace=" + namespace, "--output=yaml"}
	if err := kcommand.Execute(); err != nil {
		os.Exit(1)
	}

	fmt.Println(pod)

	/*
		selector := labels.SelectorFromSet(dc.Spec.Selector)
		//sortBy := func(pods []*v1.Pod) sort.Interface { return controller.ByLogging(pods) }
		sortBy := func(pods []*v1.Pod) sort.Interface { return sort.Reverse(controller.ActivePods(pods)) }
		pod, _, err := kcmdutil.GetFirstPod(kc.Core(), namespace, selector, time.Second*10, sortBy)
		checkErr(err)
	*/

	/*
		fmt.Printf("\n")
		os.Args = []string{"oc", "get", "all", "--all-namespaces"}
		if err := command.Execute(); err != nil {
			os.Exit(1)
		}
	*/

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

type scc struct {
	Priority int `json:"priority"`
}

type scca struct {
	Annotations map[string]string `json:"annotations,omitempty" protobuf:"bytes,12,rep,name=annotations"`
}
