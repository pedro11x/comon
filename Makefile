.PHONY := build
.DEFAULT_GOAL := build

NAME := comon
VER  := 1
IMAGE_BUILDER := docker build

#----------------Build Commands-----------------------
build:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o ${NAME}
build-arm:
	CGO_ENABLED=0 GOOS=linux GOARCH=arm go build -o ${NAME}-arm

run: vendor
	go run .

${NAME}: build
${NAME}-arm: build-arm

#----------------Docker build Commands----------------
contain: ${NAME}
	${IMAGE_BUILDER} -t ${NAME}:${VER} --build-arg NAME=${NAME} .
	#--platform linux/amd64 .
contain-arm: ${NAME}-arm
	${IMAGE_BUILDER} -t ${NAME}:${VER} --build-arg NAME=${NAME}-arm .
	#--platform linux/armhf .

container: contain
	docker run --name ${NAME} -p 9099:9099 --mount type=bind,source=/var/run/docker.sock,target=/var/run/docker.sock --rm ${NAME}:${VER}

#----------------Maintenance Commands-----------------
name:
	@echo ${NAME}
