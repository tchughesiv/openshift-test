# sccoc

[![Go Report Card](https://goreportcard.com/badge/github.com/tchughesiv/sccoc)](https://goreportcard.com/report/github.com/tchughesiv/sccoc)

[wip] openshift scc image test tool

 - relies on Origin release-3.7 as a submodule

### Getting started

The goal of this tool is to provide an easier way of testing a container against various security contexts w/o running a full OpenShift cluster. This tool only requires access to a k8s supported container runtime... docker, cri-o, etc.

`sccoc run` is the only command allowed w/ this tool today.  It maps directly to the [`oc run`](https://docs.openshift.org/latest/cli_reference/basic_cli_operations.html#run) command. Currently, only a pod resource is generated/allowed.

#### build
```shell
$ git clone https://github.com/tchughesiv/sccoc $GOPATH/src/github.com/openshift/origin
$ cd $GOPATH/src/github.com/openshift/origin/
$ make
```

#### run
```shell
# set path to sccoc binary
$ sccoc=$(source ./origin/hack/lib/init.sh && which sccoc) && echo $sccoc

# the tool defaults to the "restricted" scc... e.g.
$ sudo $sccoc run testpod --image=registry.centos.org/container-examples/starter-arbitrary-uid

# or, you can specify an alternate scc w/ the "OPENSHIFT_SCC" env variable
$ OPENSHIFT_SCC=nonroot sudo -E $sccoc run testpod --image=registry.centos.org/container-examples/starter-arbitrary-uid

# you can specify a host port, for example, using the run options... e.g. mysql on 3306
$ sudo $sccoc run mariadb --image=centos/mariadb-102-centos7 --env="MYSQL_ROOT_PASSWORD=test" --port=3306 --hostport=3306
$ telnet localhost 3306
```

It's currently helpfuly to open a separate terminal while your container deploys and monitor the runtime for your pod. Once the image is pulled and pod deployed, sccoc can be exited.

#### install
```shell
# set path to sccoc binary
$ sccoc=$(source ./origin/hack/lib/init.sh && which sccoc) && echo $sccoc

# manual install for now
$ sudo install -m755 $sccoc /usr/bin

$ sudo sccoc run testpod --image=registry.centos.org/container-examples/starter-arbitrary-uid
```

#### dev
```shell
$ cd $GOPATH/src/github.com/openshift/origin/
$ git submodule update --init
#$ git submodule add -f -b release-3.7 https://github.com/openshift/origin
#$ ln -s ./origin/vendor
#$ ln -s ./origin/pkg
#$ ln -s ./origin/test
```
