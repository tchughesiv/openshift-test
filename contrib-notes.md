## Contrib Notes

initial dev space:
```shell
cd ~
rm -rf $GOPATH 
mkdir -p $GOPATH/src/k8s.io/
cd $GOPATH/src/k8s.io/
git clone https://github.com/openshift/kubernetes
# sudo yum install mercurial git
go get -u github.com/tools/godep github.com/Masterminds/glide github.com/sgotti/glide-vc
go get -d github.com/openshift/origin
cd $GOPATH/src/github.com/openshift/origin/
git checkout v3.6.0
# ?? rm -rf $GOPATH/src/github.com/openshift/origin/vendor
godep restore
```

longterm deps:
```shell
go get -d github.com/tchughesiv/sccoc
cd $GOPATH/src/github.com/tchughesiv/sccoc/
git checkout devel
godep save ./...
# glide cache-clear
# glide update --strip-vendor
# glide up
```
