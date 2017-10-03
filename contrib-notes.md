## Contrib Notes

initial dev space:
```shell
cd ~
rm -rf $GOPATH && mkdir -p $GOPATH
git clone https://github.com/openshift/kubernetes $GOPATH/src/k8s.io/kubernetes
# sudo yum install mercurial git
go get -u github.com/tools/godep github.com/Masterminds/glide github.com/sgotti/glide-vc
go get -d github.com/openshift/origin
cd $GOPATH/src/github.com/openshift/origin/
git checkout v3.6.0
rm -rf $GOPATH/src/github.com/openshift/origin/vendor
# github.com/cloudflare/cfssl dep fix
sed -i 's/fca70798646c8689aeae5928d4ad1278ff8a3c17/db0d0650b6496bfe8061ec56a92edd32d8e75c30/g' Godeps/Godeps.json
# github.com/google/certificate-transparency dep fix
sed -i 's/a85d8bf28a950826bf6bc0693caf384ab4c6bec9/af98904302724c29aa6659ca372d41c9687de2b7/g' Godeps/Godeps.json
# github.com/skynetservices/skydns dep fix
sed -i 's/30763c4e568fe411f1663af553c063cec8879929/8211c16267029d18ccd39bbd81a5de07927cd9a9/g' Godeps/Godeps.json
# ?? ./hack/godep-restore.sh
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
