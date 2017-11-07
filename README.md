# sccoc

[![Go Report Card](https://goreportcard.com/badge/github.com/tchughesiv/sccoc)](https://goreportcard.com/report/github.com/tchughesiv/sccoc)

[wip] openshift scc image test tool

 - relies on Origin release-3.6 as a submodule

### Getting started

```shell
$ go get -d github.com/tchughesiv/sccoc
$ cd $GOPATH/src/github.com/tchughesiv/sccoc/
# $ glide up
```

dev
```shell
$ git submodule add -b release-3.6 https://github.com/openshift/origin
$ ln -s origin/vendor vendor
```
