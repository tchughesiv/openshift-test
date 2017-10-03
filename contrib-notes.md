## Contrib Notes

initial dev space:
```shell
cd ~
# sudo yum install mercurial git
rm -rf $GOPATH && mkdir -p $GOPATH
go get -u github.com/tools/godep
git clone https://github.com/openshift/kubernetes $GOPATH/src/k8s.io/kubernetes
go get -d github.com/openshift/origin
cd $GOPATH/src/github.com/openshift/origin/
git checkout v3.7.0-0.alpha.1.546.f03551f
rm -rf $GOPATH/src/github.com/openshift/origin/vendor
# k8s.io/kubernetes dep fix
# sed -i 's/fff65cf41bdeeaff9964af98450b254f3f2da553/4b31e848f77f51d5b3ed191c6f587bd53508b3f4/g' Godeps/Godeps.json
# github.com/cloudflare/cfssl dep fix
sed -i 's/fca70798646c8689aeae5928d4ad1278ff8a3c17/db0d0650b6496bfe8061ec56a92edd32d8e75c30/g' Godeps/Godeps.json
# github.com/google/certificate-transparency dep fix
sed -i 's/a85d8bf28a950826bf6bc0693caf384ab4c6bec9/af98904302724c29aa6659ca372d41c9687de2b7/g' Godeps/Godeps.json
# github.com/skynetservices/skydns dep fix
sed -i 's/30763c4e568fe411f1663af553c063cec8879929/00ade3024f047d26130abf161900e0adb72a06f1/g' Godeps/Godeps.json
# github.com/elazarl/goproxy dep fix
sed -i 's/07b16b6e30fcac0ad8c0435548e743bcf2ca7e92/c4fc26588b6ef8af07a191fcb6476387bdd46711/g' Godeps/Godeps.json
# go get k8s.io/apimachinery/pkg/apimachinery/announced \
#   k8s.io/apimachinery/pkg/api/meta \
#   k8s.io/apimachinery/pkg/api/resource \
#   k8s.io/apimachinery/pkg/apis/meta/v1 \
#   k8s.io/apimachinery/pkg/labels \
#   k8s.io/apimachinery/pkg/runtime \
#   k8s.io/client-go/discovery
./hack/godep-restore.sh
# godep restore
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
