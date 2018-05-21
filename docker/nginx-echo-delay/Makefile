all: build push

REPO=koma77
IMAGE=nginx-echo-delay

build:
		docker build --rm -t ${REPO}/${IMAGE} . -f Dockerfile.alpine

push:
		docker push ${REPO}/${IMAGE}

.PHONY: all 
