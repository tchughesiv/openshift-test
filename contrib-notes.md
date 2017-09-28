# Contrib Notes

initial dev space:
```shell
# sudo yum install mercurial git
go get -u github.com/tools/godep
go get -d k8s.io/apimachinery \
  k8s.io/client-go \
  github.com/skynetservices/skydns \
  k8s.io/kubernetes \
  github.com/openshift/origin

cd $GOPATH/src/k8s.io/kubernetes/
git checkout release-1.6
godep restore
cd $GOPATH/src/github.com/openshift/origin/
git checkout v3.6.0
pushd $GOPATH/src/k8s.io/kubernetes
git remote add openshift https://github.com/openshift/kubernetes.git
git fetch openshift
popd
./hack/godep-restore.sh 
godep restore
```

longterm deps:
```shell
go get -d github.com/tchughesiv/sccoc
cd $GOPATH/src/github.com/tchughesiv/sccoc/
git checkout devel
godep save
```
