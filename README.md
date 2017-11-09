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
$ git submodule add -f https://github.com/openshift/origin
$ cd ./origin/
$ git checkout v3.6.1 # 008f2d5528bf998326b5eb3f1fe3144c59392b9d
$ cd $GOPATH/src/github.com/tchughesiv/sccoc/
$ ln -s ./origin/vendor
# $ git submodule add -f https://github.com/openshift/kubernetes-client-go ./vendor/k8s.io/client-go
# $ git submodule add -f https://github.com/juju/ratelimit ./vendor/github.com/juju/ratelimit
# $ git submodule add -f https://github.com/openshift/kubernetes ./vendor/k8s.io/kubernetes
# $ cd ./vendor/k8s.io/kubernetes
# $ git checkout fff65cf41bdeeaff9964af98450b254f3f2da553 
```
