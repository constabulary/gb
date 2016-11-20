#!/bin/bash

set -e

rm -f coverage.txt

for d in $(go list github.com/constabulary/gb/...); do
  go test -coverprofile=profile.out -covermode=atomic $d

  if [ -f profile.out ]; then
    cat profile.out >> coverage.txt
    rm profile.out
  fi
done
