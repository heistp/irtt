#!/bin/sh

action="build"
pkg="github.com/peteheist/irtt/cmd/irtt"
ldflags="-X github.com/peteheist/irtt.Version=0.9-git-$(git describe --always --long --dirty)"
linkshared=""
tags=""
race=""
env=""

# interpret keywords
for a in $*; do
	case "$a" in
		"install") action="install"
		ldflags="$ldflags -s -w"
		;;
		"nobuild") nobuild="1"
		;;
		"man") man="1"
		;;
		"min") ldflags="$ldflags -s -w"
		;;
		"linkshared") linkshared="-linkshared"
		;;
		"race") race="-race"
		;;
		"profile") tags="$tags profile"
		;;
		"prod") tags="$tags prod"
		;;
		"linux-386"|"linux") env="GOOS=linux GOARCH=386"
		;;
		"linux-amd64"|"linux64") env="GOOS=linux GOARCH=amd64"
		;;
		"linux-arm"|"rpi") env="GOOS=linux GOARCH=arm"
		;;
		"linux-mips64"|"erl") env="GOOS=linux GOARCH=mips"
		;;
		"linux-mipsle"|"erx"|"om2p") env="GOOS=linux GOARCH=mipsle"
		;;
		"darwin-amd64"|"osx") env="GOOS=darwin GOARCH=amd64"
		;;
		"win"|"windows") env="GOOS=windows GOARCH=386"
		;;
		"win64"|"windows64") env="GOOS=windows GOARCH=amd64"
		;;
		*) echo "Unknown parameter: $a"
		exit 1
		;;
	esac
done

# build source
if [ -z "$nobuild" ]; then
	go generate
	eval $env go $action -tags \'$tags\' $race -ldflags=\'$ldflags\' $linkshared $pkg
fi

# generate man pages
if [ -n "$man" ]; then
	md2man-roff man/irtt.md > man/irtt.1
	md2man-roff man/irtt-client.md > man/irtt-client.1
	md2man-roff man/irtt-server.md > man/irtt-server.1
fi
