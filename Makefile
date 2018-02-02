WHAT=cmd/sccoc
BUILDER_IMAGE=registry.centos.org/che-stacks/centos-go
IMAGE_NAME=docker.io/tchughesiv/sccoc
OUTPUT_DIR=origin/_output/local/bin/linux/amd64
BINARY=sccoc

.PHONY: all build clean

all: build
build:
	git submodule update --init
	ln -fs ../../${WHAT} origin/cmd/
	chmod +x *build.sh
	./build.sh ${WHAT} ${BUILDER_IMAGE} ${OUTPUT_DIR} ${BINARY}
	docker build --build-arg sccoc=${BINARY} -t ${IMAGE_NAME} ${OUTPUT_DIR}/

#install:
#	$(eval SCCOC_PATH=$(source ./origin/hack/lib/init.sh && which sccoc)) \
#	echo $$SCCOC_PATH
#	install -m755 $$SCCOC_PATH /usr/bin
#.PHONY: install

clean:
	rm -rf origin/_output origin/${WHAT}
