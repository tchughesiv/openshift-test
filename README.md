# sccoc

[![Go Report Card](https://goreportcard.com/badge/github.com/tchughesiv/sccoc)](https://goreportcard.com/report/github.com/tchughesiv/sccoc)

[wip] openshift scc image test tool

 - relies on Origin release-3.7 as a submodule

### Getting started

The goal of this tool is to provide an easier way of testing a container against various security contexts w/o running a full OpenShift cluster. This tool only requires access to a k8s supported container runtime... docker, cri-o, etc.

`sccoc run` is the only command allowed w/ this tool today.  It maps directly to the [`oc run`](https://docs.openshift.org/latest/cli_reference/basic_cli_operations.html#run) command. Currently, only a pod resource is generated/allowed.

build
```shell
$ git clone https://github.com/tchughesiv/sccoc $GOPATH/src/github.com/openshift/origin
$ cd $GOPATH/src/github.com/openshift/origin/
$ make
# set path to sccoc binary
$ export PATH=$(source ./origin/hack/lib/init.sh && echo $PATH)
# it defaults to the "restricted" scc... e.g.
$ sudo sccoc run testpod --image=registry.centos.org/container-examples/starter-arbitrary-uid
# you can specify an alternate scc w/ the "OPENSHIFT_SCC" env variable
$ sudo OPENSHIFT_SCC=nonroot sccoc run testpod --image=registry.centos.org/container-examples/starter-arbitrary-uid
# you can specify a host port, for example, using the run options
$ sudo OPENSHIFT_SCC=nonroot sccoc run testpod --image=registry.centos.org/container-examples/starter-arbitrary-uid
```

It's currently helpfuly to open a separate terminal while your container deploys and monitor the runtime for your pod. Once the image is pulled and pod deployed, sccoc can be exited.

dev
```shell
$ git submodule update --init
#$ git submodule add -f -b release-3.7 https://github.com/openshift/origin
#$ ln -s ./origin/vendor
#$ ln -s ./origin/pkg
#$ ln -s ./origin/test
```
