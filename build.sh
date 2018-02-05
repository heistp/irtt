#!/bin/sh

# This script may be used during development for making builds and generating doc.
# Requirements:
# - stringer (go get -u -a golang.org/x/tools/cmd/stringer)
# - pandoc (apt-get install pandoc OR brew install pandoc)

action="build"
pkg="github.com/peteheist/irtt/cmd/irtt"
ldflags=""
linkshared=""
tags=""
race=""
env=""

# html filter
html_filter() {
	sed 's/<table>/<table class="pure-table pure-table-striped">/g'
}

# interpret keywords
for a in $*; do
	case "$a" in
		"install") action="install"
		ldflags="$ldflags -s -w"
		;;
		"nobuild") nobuild="1"
		;;
		"nodoc") nodoc="1"
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

# generate docs
if [ -z "$nodoc" ]; then
	for f in irtt irtt-client irtt-server; do
		pandoc -s -t man doc/$f.md -o doc/$f.1
		pandoc -t html -H doc/head.html doc/$f.md | html_filter > doc/$f.html
	done
fi
