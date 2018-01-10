#!/bin/sh
set -x
cd $(dirname $(readlink -f $0))

##if [ -L /etc/redhat-release ]; then
##    yum -y install golang
##else
##    yum -y install --disablerepo "*" --enablerepo rhel-7-server-rpms,rhel-7-server-optional-rpms golang
##fi

GO_BUILD=$1
#export GOPATH=${HOME}/go
mkdir -p ${GOPATH}/src/github/openshift
ln -fs /build ${GOPATH}/src/github/openshift/origin

#CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main ${GO_BUILD}
#go-md2man -in help.md -out help.1
make -C ${GOPATH}/src/github/openshift/origin/origin WHAT=${GO_BUILD}
