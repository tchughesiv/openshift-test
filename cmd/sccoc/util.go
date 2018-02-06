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

func contains(o []string, f string) bool {
	for _, a := range o {
		if a == f {
			return true
		}
	}
	return false
}

// func recreatePod(kclient internalclientset.Interface, namespace string, mpath string) {
func recreatePod(kclient internalclientset.Interface, namespace string) {
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
}

func runKubelet(nodeconfig *node.NodeConfig) {
	//kubeCfg := s.KubeletConfiguration
	kubeDeps := nodeconfig.KubeletDeps
	s := nodeconfig.KubeletServer
	dinfo, err := kubeDeps.DockerClient.Info()
	checkErr(err)
	s.CgroupDriver = dinfo.CgroupDriver
	s.Containerized = true
	s.RunOnce = false
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
	p.ObjectMeta.ResourceVersion = ""
	p.Spec.ServiceAccountName = ""
	automountSaToken := false
	p.Spec.AutomountServiceAccountToken = &automountSaToken
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

func sliceInsert(slice []string, index int, value string) []string {
	s := append(slice, "")
	copy(s[index+1:], s[index:])
	s[index] = value
	return s
}
