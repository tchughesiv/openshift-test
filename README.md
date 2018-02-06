# sccoc

[![Go Report Card](https://goreportcard.com/badge/github.com/tchughesiv/sccoc)](https://goreportcard.com/report/github.com/tchughesiv/sccoc)

[wip] openshift scc image test tool

 - relies on Origin v3.7.x as a submodule

### Getting started

The goal of this tool is to provide an easier way of testing a container against various security contexts w/o running a full OpenShift cluster. This tool only requires access to a k8s supported container runtime... docker, cri-o, etc.

`sccoc run` is the only command allowed w/ this tool today.  It maps directly to the [`oc run`](https://docs.openshift.org/latest/cli_reference/basic_cli_operations.html#run) command. Currently, only a pod resource is generated/allowed.

#### run the image

For macOS users... you should also set `BASETMPDIR=/var/lib` prior to running.

```shell
# create an alias to sccoc image
$ alias sccoc='_(){ export S=${OPENSHIFT_SCC} V=${BASETMPDIR:-${HOME}/openshift-sccoc}; }; _; docker run --rm --privileged --pid=host --net=host -v /:/rootfs:ro -v /dev:/dev -v /var/run:/var/run -v /sys:/sys -v /sys/fs/cgroup:/sys/fs/cgroup:ro -v /sys/devices/virtual/net:/sys/devices/virtual/net -v /var/lib/docker:/var/lib/docker -v ${V}/openshift.local.config:${V}/openshift.local.config -v ${V}/volume:${V}/volume:rslave -e OPENSHIFT_SCC=${S} -e BASETMPDIR=${V} docker.io/tchughesiv/sccoc'

# the tool defaults to the "restricted" scc... e.g.
$ docker pull registry.centos.org/container-examples/starter-arbitrary-uid
$ sccoc run testpod --image=registry.centos.org/container-examples/starter-arbitrary-uid

# or, you can specify an alternate scc w/ the "OPENSHIFT_SCC" env variable
$ OPENSHIFT_SCC=nonroot
$ sccoc run mariadb --image=docker.io/centos/mariadb-102-centos7 --env MYSQL_ROOT_PASSWORD=test

# you can specify a host port, for example, using the run options... e.g. mysql on 3306
$ OPENSHIFT_SCC=anyuid
$ sccoc run nginx --image=docker.io/nginx --port=80 --hostport=8080
$ curl localhost:8080
```

It's currently helpfuly to open a separate terminal while your container deploys and monitor the runtime for your pod. Once the image is pulled and pod deployed, sccoc can be exited.

Running a container from the Red Hat Container Catalog:
```shell
$ docker login registry.connect.redhat.com
$ docker pull registry.connect.redhat.com/crunchydata/crunchy-postgres
$ OPENSHIFT_SCC=nonroot # or OPENSHIFT_SCC=anyuid
$ sccoc run crunchydb --image=registry.connect.redhat.com/crunchydata/crunchy-postgres --env PG_MODE=primary --env PG_PRIMARY_USER=admin --env PG_PRIMARY_PASSWORD=pw --env PG_USER=user --env PG_PASSWORD=pw --env PG_DATABASE=db --env PG_ROOT_PASSWORD=pw --env PG_PRIMARY_PORT=5432
```

#### build
```shell
$ git clone https://github.com/tchughesiv/sccoc $GOPATH/src/github.com/openshift/origin
$ cd $GOPATH/src/github.com/openshift/origin/
$ make
```

#### dev
```shell
$ cd $GOPATH/src/github.com/openshift/origin/
$ git submodule update --init
#$ git submodule add -f -b v3.7.1 https://github.com/openshift/origin
#$ ln -s ./origin/vendor
#$ ln -s ./origin/pkg
#$ ln -s ./origin/test
```
