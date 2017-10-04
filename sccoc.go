package main

import (
	"fmt"
	"log"

	bp "github.com/openshift/origin/pkg/cmd/server/bootstrappolicy"
	securityapi "github.com/openshift/origin/pkg/security/apis/security"
	// oscc "github.com/openshift/origin/pkg/security/scc"
	scc "github.com/openshift/origin/pkg/security/securitycontextconstraints"
	kapi "k8s.io/kubernetes/pkg/api"
)

// const defaultScc = "anyuid"
const defaultScc = "restricted"

func main() {
	var sccn *securityapi.SecurityContextConstraints
	//	ns := testing.CreateNamespaceForTest()
	//	ns.Name = util.RandomNamespace("tmp")
	groups, users := bp.GetBoostrapSCCAccess(bp.DefaultOpenShiftInfraNamespace)
	bootstrappedConstraints := bp.GetBootstrapSecurityContextConstraints(groups, users)
	for _, v := range bootstrappedConstraints {
		if v.Name == defaultScc {
			sccn = validNewSecurityContextConstraints(v)
		}
	}

	sccn.RunAsUser.UIDRangeMin = &(&struct{ x int64 }{1000100000}).x
	sccn.RunAsUser.UIDRangeMax = &(&struct{ x int64 }{1000110000}).x
	sccn.SELinuxContext.SELinuxOptions = &kapi.SELinuxOptions{Level: "s9:z0,z1"}

	fmt.Printf("%#v\n\n", sccn.Name)
	fmt.Printf("%#v\n\n", sccn.SELinuxContext.SELinuxOptions)

	_, err := scc.NewSimpleProvider(sccn)
	checkErr(err)
}

// convert specified scc definition into container runtime configs - using origin code
// run image accordingly directly against container runtime... no ocp/k8s involvement

func validNewSecurityContextConstraints(sccn securityapi.SecurityContextConstraints) *securityapi.SecurityContextConstraints {
	return &sccn
}

func checkErr(err error) {
	if err != nil {
		log.Println(err)
	}
}
