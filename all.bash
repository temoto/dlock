#!/bin/bash
set -e

cmd=${*-test}

( cd dlock; protoc --go_out=./ *.proto )

if [[ "$cmd" == "test" ]] ; then
  rm -f coverage.txt
  for d in $(go list ./... |fgrep -v vendor) ; do
  	tmp=/tmp/coverage-one.txt
    go test -race -coverprofile=$tmp -covermode=atomic $d
    [[ -f "$tmp" ]] && cat $tmp >>coverage.txt
  done
else
  go $cmd ./...
fi
