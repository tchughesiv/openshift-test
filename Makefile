WHAT=cmd/sccoc

all: build
build:
	git submodule update --init
	ln -fs ../../${WHAT} origin/cmd/
	${MAKE} -C origin WHAT=${WHAT}
.PHONY: build

#install:
#	$(eval SCCOC_PATH=$(source ./origin/hack/lib/init.sh && which sccoc)) \
#	echo $$SCCOC_PATH
#	install -m755 $$SCCOC_PATH /usr/bin
#.PHONY: install

clean:
	rm -rf origin/_output origin/${WHAT}
.PHONY: clean
