#!/bin/bash
set -e

cmd=${*-test}

( cd dlock; protoc --go_out=./ *.proto )

go $cmd ./...
