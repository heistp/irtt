#
# Makefile
#

BINARY:=irtt

all: clean build version

test:
	go test

clean:
	[ -f ${BINARY} ] && /bin/rm -rf ./${BINARY} || true

build:
	CGO_ENABLED=0 go build -o ./${BINARY} ./${BINARY}.go

version:
	./${BINARY} help

# https://words.filippo.io/shrink-your-go-binaries-with-this-one-weird-trick/
buildsmall:
	CGO_ENABLED=0 go build -ldflags "-s -w" -o ./${BINARY} ./${BINARY}.go

shrink:
	upx --brute ./${BINARY}
