# sccoc

[![Go Report Card](https://goreportcard.com/badge/github.com/tchughesiv/sccoc)](https://goreportcard.com/report/github.com/tchughesiv/sccoc)

[wip] openshift scc image test tool

 - relies on Origin release-3.6 as a submodule

### Getting started

```shell
$ go get -d github.com/tchughesiv/sccoc
$ mkdir -p $GOPATH/src/github.com/openshift
$ ln -s $GOPATH/src/github.com/tchughesiv/sccoc $GOPATH/src/github.com/openshift/origin
$ cd $GOPATH/src/github.com/tchughesiv/sccoc/
$ git submodule update --init
$ go build
# $ go run sccoc.go
```

dev
```shell
$ git submodule add -f -b release-3.6 https://github.com/openshift/origin
$ ln -s ./origin/vendor
$ ln -s ./origin/pkg
$ ln -s ./origin/test
```
