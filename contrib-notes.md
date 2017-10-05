## Contrib Notes

```shell
# sudo yum install mercurial git
go get -u github.com/Masterminds/glide # github.com/sgotti/glide-vc github.com/alecthomas/gometalinter
# go get -u github.com/golang/dep
go get -d github.com/tchughesiv/sccoc
cd $GOPATH/src/github.com/tchughesiv/sccoc/
git checkout devel
glide install
# gometalinter -i -u
# glide install
# glide up -v
# glide up

#rm -rf $GOPATH/pkg/dep/sources vendor*
#rm -rf ./Gopkg.* ./vendor* ./_vendor*
#dep init
#dep ensure -v

# before commit/push
# glide-vc
```
