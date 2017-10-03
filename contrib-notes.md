## Contrib Notes

initial dev space:
```shell
# sudo yum install mercurial git
go get -u github.com/tools/godep github.com/jteeuwen/go-bindata/go-bindata github.com/Masterminds/glide
go get -d github.com/tchughesiv/sccoc
cd $GOPATH/src/github.com/tchughesiv/sccoc/
git checkout devel
glide install
```
