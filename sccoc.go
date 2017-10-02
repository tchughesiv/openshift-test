package main

import (
	"fmt"
	"log"

	bp "github.com/openshift/origin/pkg/cmd/server/bootstrappolicy"
	"github.com/openshift/origin/pkg/security/admission/testing"
	securityapi "github.com/openshift/origin/pkg/security/apis/security"
	"github.com/openshift/origin/pkg/security/securitycontextconstraints"
	"github.com/openshift/origin/test/util"
//	kapi "k8s.io/kubernetes/pkg/api"
)

// const defaultScc = "anyuid"
const defaultScc = "restricted"

func main() {
	var scc *securityapi.SecurityContextConstraints
	ns := testing.CreateNamespaceForTest()
	ns.Name = util.RandomNamespace("tmp")
	groups, users := bp.GetBoostrapSCCAccess(ns.Name)
	bootstrappedConstraints := bp.GetBootstrapSecurityContextConstraints(groups, users)
	for _, v := range bootstrappedConstraints {
		if v.Name == defaultScc {
			scc = v
			scc.RunAsUser.UIDRangeMin = &(&struct{ x int64 }{1000100000}).x
			scc.RunAsUser.UIDRangeMax = &(&struct{ x int64 }{1000110000}).x
		}
	}

/*
	scc.SELinuxContext.SELinuxOptions = &kapi.SELinuxOptions{
		Level: "s9:z0,z1",
	}
*/

    fmt.Printf("%#v\n\n", scc.Name)
	fmt.Printf("%#v\n\n", scc)

	_, err := securitycontextconstraints.NewSimpleProvider(scc)
	checkErr(err)
}


// convert specified scc definition into container runtime configs - using origin code
// run image accordingly directly against container runtime... no ocp/k8s involvement

/*
	func validNewSecurityContextConstraints(scc securityapi.SecurityContextConstraints) *securityapi.SecurityContextConstraints {
		return &scc
	}
*/

func checkErr(err error) {
	if err != nil {
		log.Println(err)
	}
}
