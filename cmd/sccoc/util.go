package main

import (
	"encoding/json"
	"log"
	"os"

	authorizationapi "github.com/openshift/origin/pkg/authorization/apis/authorization"
	bp "github.com/openshift/origin/pkg/cmd/server/bootstrappolicy"
	"github.com/openshift/origin/pkg/oc/admin/policy"
	securityclientinternal "github.com/openshift/origin/pkg/security/generated/internalclientset"
	"k8s.io/apimachinery/pkg/types"
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
		err = o.AddSCC()
		checkErr(err)
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
		err := o.RemoveSCC()
		checkErr(err)
	}
	ch <- true
}
