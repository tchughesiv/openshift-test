package main

import (
	"encoding/json"
	"log"
	"os"

	authorizationapi "github.com/openshift/origin/pkg/authorization/apis/authorization"
	bp "github.com/openshift/origin/pkg/cmd/server/bootstrappolicy"
	"github.com/openshift/origin/pkg/cmd/server/kubernetes/node"
	"github.com/openshift/origin/pkg/oc/admin/policy"
	securityclientinternal "github.com/openshift/origin/pkg/security/generated/internalclientset"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/kubernetes/cmd/kubelet/app"
	"k8s.io/kubernetes/pkg/api"
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

//func recreatePod(kclient internalclientset.Interface, namespace string, mpath string) v1.Pod {
func recreatePod(kclient internalclientset.Interface, namespace string, mpath string) {
	zero := int64(0)
	do := metav1.DeleteOptions{GracePeriodSeconds: &zero}

	podint := kclient.Core().Pods(namespace)
	podl, err := podint.List(metav1.ListOptions{})
	checkErr(err)
	pod, err := podint.Get(podl.Items[0].GetName(), metav1.GetOptions{})
	checkErr(err)

	// modify pod
	modPod(pod)

	// remove pod secrets
	rmSV(pod)

	// delete pod
	checkErr(podint.Delete(pod.Name, &do))

	// recreate modified pod w/o secret volume(s)
	_, err = podint.Create(pod)
	checkErr(err)

	/*
		jp, err := json.Marshal(n)
		checkErr(err)

		fmt.Printf("\n")
		fmt.Println(string(jp))
	*/

	// convert pod mods

	/*
			externalPod := &v1.Pod{}
			checkErr(v1.Convert_api_Pod_To_v1_Pod(pod, externalPod, nil))
			p := *externalPod
				podyf := mpath + "/" + p.Name + ".yaml"

				//	u := string(p.ObjectMeta.UID)
				//	podyf := mpath + "/" + u + ".yaml"
				//	p.Name = u
				//	p.SelfLink = "/api/" + p.TypeMeta.APIVersion + "/namespaces/" + p.Namespace + "/pods/" + p.Name

				p.Status = v1.PodStatus{}
				p.TypeMeta.Kind = "Pod"
				p.TypeMeta.APIVersion = "v1"
				p.Spec.DeprecatedServiceAccount = ""

				jpod, err := json.Marshal(p)
				checkErr(err)
				pyaml, err := yaml.JSONToYAML(jpod)
				checkErr(err)

				ioutil.WriteFile(podyf, pyaml, os.FileMode(0644))
		return p
	*/
}

//func runKubelet(nodeconfig *node.NodeConfig, p v1.Pod) {
func runKubelet(nodeconfig *node.NodeConfig) {
	kubeDeps := nodeconfig.KubeletDeps
	s := nodeconfig.KubeletServer
	s.Containerized = true

	dinfo, err := kubeDeps.DockerClient.Info()
	checkErr(err)
	s.CgroupDriver = dinfo.CgroupDriver
	s.RunOnce = false
	//kubeCfg := s.KubeletConfiguration

	checkErr(app.Run(s, kubeDeps))
	//checkErr(app.RunKubelet(&kubeFlags, &kubeCfg, kubeDeps, false, true))
}

func mkDir(dir string) {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		os.Mkdir(dir, 0755)
	}
}

func sccMod(sflag string, namespace string, securityClient securityclientinternal.Interface) {
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
}

func sccRm(sflag string, namespace string, securityClient securityclientinternal.Interface) {
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
}

func modPod(p *api.Pod) {
	p.Status = api.PodStatus{}
	//pn.UID = ""
	p.ObjectMeta.ResourceVersion = ""
	p.Spec.ServiceAccountName = ""
	//pod.Spec.DeprecatedServiceAccount = ""
	//pn.Spec.SchedulerName = ""
	//pod.Spec.ImagePullSecrets = []v1.LocalObjectReference{}
	automountSaToken := false
	p.Spec.AutomountServiceAccountToken = &automountSaToken
	//pn.Spec.DNSPolicy = api.DNSDefault
}

func rmSV(p *api.Pod) {
	for i, v := range p.Spec.Volumes {
		if v.Secret != nil {
			for n, c := range p.Spec.Containers {
				for x, m := range c.VolumeMounts {
					if m.Name == v.Name {
						p.Spec.Containers[n].VolumeMounts = append(p.Spec.Containers[n].VolumeMounts[:x], p.Spec.Containers[n].VolumeMounts[x+1:]...)
					}
				}
			}
			p.Spec.Volumes = append(p.Spec.Volumes[:i], p.Spec.Volumes[i+1:]...)
		}
	}
}
