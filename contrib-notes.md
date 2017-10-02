## Contrib Notes

initial dev space:
```shell
# sudo yum install mercurial git
go get -u github.com/tools/godep github.com/Masterminds/glide github.com/sgotti/glide-vc
go get -d github.com/openshift/origin
cd $GOPATH/src/github.com/openshift/origin/
git checkout v3.6.0
# ?? rm -rf $GOPATH/src/github.com/openshift/origin/vendor
make clean ; godep restore
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
