package main

import (
	"fmt"
	"log"

	bp "github.com/openshift/origin/pkg/cmd/server/bootstrappolicy"
	"github.com/openshift/origin/pkg/security/admission/testing"
	securityapi "github.com/openshift/origin/pkg/security/apis/security"
	"github.com/openshift/origin/pkg/security/securitycontextconstraints"
	"github.com/openshift/origin/test/util"
	kapi "k8s.io/kubernetes/pkg/api"
)

// const defaultScc = "anyuid"
const defaultScc = "restricted"

func main() {
	var scc securityapi.SecurityContextConstraints
	ns := testing.CreateNamespaceForTest()
	ns.ObjectMeta.Name = util.RandomNamespace("tmp")
	fmt.Printf("\n%#v\n\n", ns.Name)
	fmt.Printf("%#v\n\n", ns.ObjectMeta.Annotations)

	//groups, users := bp.GetBoostrapSCCAccess(bp.DefaultOpenShiftInfraNamespace)
	groups, users := bp.GetBoostrapSCCAccess(ns.Name)
	bootstrappedConstraints := bp.GetBootstrapSecurityContextConstraints(groups, users)

	for _, v := range bootstrappedConstraints {
		if v.Name == defaultScc {
			scc = v
		}
	}
	
	sccn := validNewSecurityContextConstraints(scc)
	sccn.RunAsUser.UIDRangeMin = &(&struct{ x int64 }{1000100000}).x
	sccn.RunAsUser.UIDRangeMax = &(&struct{ x int64 }{1000110000}).x
	sccn.SELinuxContext.SELinuxOptions = &kapi.SELinuxOptions{
		Level: "s0:c10,c5",
	}

	fmt.Printf("%#v\n\n", sccn.Name)
	fmt.Printf("%#v\n\n", sccn)

	_, err := securitycontextconstraints.NewSimpleProvider(sccn)
	checkErr(err)
}

// convert specified scc definition into container runtime configs - using origin code
// run image accordingly directly against container runtime... no ocp/k8s involvement

type intwrapper struct{ x int64 }

func validNewSecurityContextConstraints(scc securityapi.SecurityContextConstraints) *securityapi.SecurityContextConstraints {
	return &scc
}

func checkErr(err error) {
	if err != nil {
		log.Println(err)
	}
}
