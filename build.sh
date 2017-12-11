#!/bin/sh

action="build"
pkgbase="github.com/peteheist/irtt"
pkgs="$pkgbase/cmd/irtt"
ldflags="-X github.com/peteheist/irtt.Version=0.9-git-$(git describe --always --long --dirty) -X github.com/peteheist/irtt.BuildDate=$(date -u +"%Y-%m-%dT%H:%M:%SZ")"
linkshared=""
tags=""
race=""
env=""

for a in $*; do
	case "$a" in
		"install") action="install"
		ldflags="$ldflags -s -w"
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

go generate

for p in $pkgs; do
	eval $env go $action -tags \'$tags\' $race -ldflags=\'$ldflags\' $linkshared $p
done
