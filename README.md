# sccoc

[![Go Report Card](https://goreportcard.com/badge/github.com/tchughesiv/sccoc)](https://goreportcard.com/report/github.com/tchughesiv/sccoc)

[wip] openshift scc image test tool

 - relies on Origin release-3.7 as a submodule

### Getting started

build
```shell
$ git clone https://github.com/tchughesiv/sccoc $GOROOT/src/github.com/openshift/origin
# alternatively could... 
# $ go get -d github.com/tchughesiv/sccoc
# $ mkdir -p $GOROOT/src/github.com/openshift
# $ ln -s $GOROOT/src/github.com/tchughesiv/sccoc $GOROOT/src/github.com/openshift/origin
$ cd $GOROOT/src/github.com/openshift/origin/
$ git submodule update --init
$ make -C origin WHAT=cmd/sccoc
```

dev
```shell
$ git submodule add -f -b release-3.7 https://github.com/openshift/origin
$ ln -s ./origin/vendor
$ ln -s ./origin/pkg
$ ln -s ./origin/test
```
