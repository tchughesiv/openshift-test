package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"

	"github.com/ghodss/yaml"
	authorizationapi "github.com/openshift/origin/pkg/authorization/apis/authorization"
	bp "github.com/openshift/origin/pkg/cmd/server/bootstrappolicy"
	"github.com/openshift/origin/pkg/cmd/server/kubernetes/node"
	"github.com/openshift/origin/pkg/oc/admin/policy"
	securityclientinternal "github.com/openshift/origin/pkg/security/generated/internalclientset"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/kubernetes/cmd/kubelet/app"
	kubeletoptions "k8s.io/kubernetes/cmd/kubelet/app/options"
	api "k8s.io/kubernetes/pkg/api"
	v1 "k8s.io/kubernetes/pkg/api/v1"
	"k8s.io/kubernetes/pkg/client/clientset_generated/internalclientset"
)

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

func exportPod(kclient internalclientset.Interface, namespace string, mpath string) (*v1.Pod, *[]api.Pod) {
	fmt.Printf("\n")
	podint := kclient.Core().Pods(namespace)
	podl, err := podint.List(metav1.ListOptions{})
	checkErr(err)
	pod, err := podint.Get(podl.Items[0].GetName(), metav1.GetOptions{})
	checkErr(err)

	// mirror pod mods
	podyf := mpath + "/" + pod.Name + "-pod.yaml"
	externalPod := &v1.Pod{}
	checkErr(v1.Convert_api_Pod_To_v1_Pod(pod, externalPod, nil))
	p := *externalPod
	p.TypeMeta.Kind = "Pod"
	p.TypeMeta.APIVersion = "v1"
	p.ObjectMeta.UID = ""
	p.ObjectMeta.ResourceVersion = ""
	p.Spec.ServiceAccountName = ""
	p.Spec.DeprecatedServiceAccount = ""
	//automountSaToken := false
	//p.Spec.AutomountServiceAccountToken = &automountSaToken
	for i, v := range p.Spec.Volumes {
		if v.Secret != nil {
			for n, c := range p.Spec.Containers {
				for x, m := range c.VolumeMounts {
					if m.Name == v.Name {
						fmt.Println("\n" + m.Name + "\n")
						p.Spec.Containers[n].VolumeMounts = append(p.Spec.Containers[n].VolumeMounts[:x], p.Spec.Containers[n].VolumeMounts[x+1:]...)
					}
				}
			}
			p.Spec.Volumes = append(p.Spec.Volumes[:i], p.Spec.Volumes[i+1:]...)
		}
	}
	jpod, err := json.Marshal(p)
	checkErr(err)
	pyaml, err := yaml.JSONToYAML(jpod)
	checkErr(err)
	ioutil.WriteFile(podyf, pyaml, os.FileMode(0644))

	return &p, &podl.Items
}

func runKubelet(s *kubeletoptions.KubeletServer, nodeconfig *node.NodeConfig) {
	// requires higher max user watches for file method...
	// sudo sysctl fs.inotify.max_user_watches=524288
	// ?? make the change permanent, edit the file /etc/sysctl.conf and add the line to the end of the file
	// remove serviceaccount, secrets, resourceVersion from pod yaml before processing as mirror pod

	// s.RunOnce = true
	checkErr(app.Run(s, nodeconfig.KubeletDeps))
}

func mkDir(dir string) {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		os.Mkdir(dir, 0755)
	}
}

func sccMod(sflag string, namespace string, securityClient securityclientinternal.Interface, ch chan<- bool) {
	if sflag != bp.SecurityContextConstraintRestricted && sflag != bp.SecurityContextConstraintsAnyUID {
		sa := "system:serviceaccount:" + namespace + ":" + bp.DefaultServiceAccountName
		patch, err := json.Marshal(scc{Priority: 1})
		checkErr(err)
		_, err = securityClient.Security().SecurityContextConstraints().Patch(sflag, types.StrategicMergePatchType, patch, "")
		checkErr(err)

		o := &policy.SCCModificationOptions{}
		o.Out = os.Stdout
		o.SCCName = sflag
		o.Subjects = authorizationapi.BuildSubjects([]string{sa}, []string{})
		o.SCCInterface = securityClient.Security().SecurityContextConstraints()
		o.DefaultSubjectNamespace = namespace
		checkErr(o.AddSCC())
	}
	ch <- true
}

func sccRm(sflag string, namespace string, securityClient securityclientinternal.Interface, ch chan<- bool) {
	if sflag != bp.SecurityContextConstraintsAnyUID {
		o := &policy.SCCModificationOptions{}
		o.Out = os.Stdout
		o.IsGroup = true
		o.SCCName = bp.SecurityContextConstraintsAnyUID
		o.Subjects = authorizationapi.BuildSubjects([]string{}, []string{"system:cluster-admins"})
		o.SCCInterface = securityClient.Security().SecurityContextConstraints()
		o.DefaultSubjectNamespace = namespace
		checkErr(o.RemoveSCC())
	}
	ch <- true
}
