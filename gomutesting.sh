#!/bin/bash

function sanityCheck() {
    if (( $(find . -iname '*.tmp' | wc -l | awk '{ print $1 }') > 0 )) ; then
        echo -e "\nWARNING - go-mutesting has not restored some files:" && find . -iname '*.tmp'
        exit 1
    fi
}

sanityCheck

tput sc ; echo -n -e "Go getting go-mutesting..." ; tput rc
go get -t -v github.com/zimmski/go-mutesting/...

tput sc ; echo -n -e "Running go-mutesting...   " ; tput rc
go-mutesting --verbose --exec-timeout=3 --blacklist=gomutesting.blacklist "${GOPATH}"/src/github.com/seborama/fuego/... 2>&1 | grep -Ev "^PASS |^Mutate |^Enable |^SKIP "

sanityCheck

