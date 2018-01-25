#!/bin/sh
set -x
cd $(dirname $(readlink -f $0))
GO_BUILD=$1
BUILDER_IMAGE=$2
OUTPUT_DIR=$3
BINARY=$4
BDIR=/build
docker run --rm -u `id -u` -tiv ${PWD}:${BDIR}:z ${BUILDER_IMAGE} \
  bash -c "${BDIR}/go_build.sh ${GO_BUILD}"
cp -p Dockerfile ${OUTPUT_DIR}/
