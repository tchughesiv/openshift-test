WHAT = cmd/sccoc

all: build
build:
	git submodule update --init
	ln -fs ../../${WHAT} origin/cmd/
	${MAKE} -C origin WHAT=${WHAT}

clean:
	rm -rf origin/_output origin/${WHAT}
.PHONY: clean
