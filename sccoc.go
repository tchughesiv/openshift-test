package main

import (
	"fmt"
	"log"

	"github.com/openshift/origin/pkg/client/testclient"
	bp "github.com/openshift/origin/pkg/cmd/server/bootstrappolicy"
	"github.com/openshift/origin/pkg/security/admission/testing"
	securityapi "github.com/openshift/origin/pkg/security/apis/security"
	"github.com/openshift/origin/pkg/security/scc"
	testutil "github.com/openshift/origin/test/util"
)

//const defaultScc = "anyuid"
const defaultScc = "restricted"

func main() {
	var sccn *securityapi.SecurityContextConstraints
	//nsa := testing.CreateSAForTest()
	ns := testing.CreateNamespaceForTest()
	ns.Name = testutil.RandomNamespace("tmp")

	groups, users := bp.GetBoostrapSCCAccess(ns.Name)
	bootstrappedConstraints := bp.GetBootstrapSecurityContextConstraints(groups, users)
	for _, v := range bootstrappedConstraints {
		if v.Name == defaultScc {
			sccn = validNewSecurityContextConstraints(v)
			sccn.RunAsUser.UIDRangeMin = &(&struct{ x int64 }{1000100000}).x
			sccn.RunAsUser.UIDRangeMax = &(&struct{ x int64 }{1000110000}).x
		}
	}

	_, kc := testclient.NewFixtureClients()
	sccp, _, err := scc.CreateProviderFromConstraint(ns.Name, ns, sccn, kc)
	checkErr(err)

	fmt.Printf("\n%#v\n\n", sccp.GetSCCName())
	fmt.Printf("%#v\n\n", sccp)
}

func validNewSecurityContextConstraints(sccn securityapi.SecurityContextConstraints) *securityapi.SecurityContextConstraints {
	return &sccn
}

func checkErr(err error) {
	if err != nil {
		log.Println(err)
	}
}

// convert specified scc definition into container runtime configs - using origin code
// run image accordingly directly against container runtime... no ocp/k8s involvement
