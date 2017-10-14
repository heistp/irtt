#!/bin/sh

action="build"
pkgbase="github.com/peteheist/irtt"
pkgs="$pkgbase/cmd/irtt"
ldflags="-X github.com/peteheist/irtt.Version=0.1`date -u +.%Y%m%d.%H%M%S`"
linkshared=""
tags=""
race=""
env=""

for a in $*; do
	case "$a" in
		"install") action="install"
		ldflags="-s -w"
		;;
		"min") ldflags="-s -w"
		;;
		"linkshared") linkshared="-linkshared"
		;;
		"race") race="-race"
		;;
		"profile") tags="$tags profile"
		;;
		"prod") tags="$tags prod"
		;;
		"openbsd-amd64") env="GOOS=openbsd GOARCH=amd64"
		;;
		"freebsd-amd64") env="GOOS=freebsd GOARCH=amd64"
		;;
		"linux-amd64") env="GOOS=linux GOARCH=amd64"
		;;
		"linux-arm"|"rpi") env="GOOS=linux GOARCH=arm"
		;;
		"win"|"windows") env="GOOS=windows GOARCH=386"
		;;
		"win64"|"windows64") env="GOOS=windows GOARCH=amd64"
		;;
		"linux-mipsle"|"erx"|"om2p") env="GOOS=linux GOARCH=mipsle"
		;;
		*) echo "Unknown parameter: $a"
		exit 1
		;;
	esac
done

go generate

for p in $pkgs; do
	eval $env go $action -tags \'$tags\' $race -ldflags=\"$ldflags\" $linkshared $p
done
