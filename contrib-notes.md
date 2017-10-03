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

dependencies to fix???
```
# cd /home/tohughes/Documents/Workspace/go_path/src/github.com/cloudflare/cfssl; git checkout fca70798646c8689aeae5928d4ad1278ff8a3c17
fatal: reference is not a tree: fca70798646c8689aeae5928d4ad1278ff8a3c17
godep: error downloading dep (github.com/cloudflare/cfssl/auth): exit status 128
# cd /home/tohughes/Documents/Workspace/go_path/src/github.com/google/certificate-transparency; git checkout a85d8bf28a950826bf6bc0693caf384ab4c6bec9
fatal: reference is not a tree: a85d8bf28a950826bf6bc0693caf384ab4c6bec9
godep: error downloading dep (github.com/google/certificate-transparency/go): exit status 128
# cd /home/tohughes/Documents/Workspace/go_path/src/github.com/skynetservices/skydns; git checkout 30763c4e568fe411f1663af553c063cec8879929
fatal: reference is not a tree: 30763c4e568fe411f1663af553c063cec8879929
godep: error downloading dep (github.com/skynetservices/skydns/cache): exit status 128
godep: Error downloading some deps. Aborting restore and check.
```
