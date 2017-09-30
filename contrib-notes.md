## Contrib Notes

initial dev space:
```shell
# sudo yum install mercurial git
go get -u github.com/tools/godep
go get -d github.com/openshift/origin k8s.io/kubernetes/pkg/api
cd $GOPATH/src/k8s.io/kubernetes/pkg/api/
git checkout v1.7.0
cd $GOPATH/src/github.com/openshift/origin/
git checkout v3.6.0
```

longterm deps:
```shell
go get -d github.com/tchughesiv/sccoc
cd $GOPATH/src/github.com/tchughesiv/sccoc/
git checkout devel
godep save
```
