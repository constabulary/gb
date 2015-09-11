#!/bin/bash
env GB_VERSION=`git describe | sed 's/-.*//' | tr -d 'v'` bash -c 'sed -i .bk "s/version := \"[^\"]*\"/version := \"$GB_VERSION\"/g" `find . -name version.go`'
