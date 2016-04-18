#!/bin/bash

go get github.com/tools/godep
go get github.com/onsi/ginkgo/ginkgo
go get github.com/onsi/gomega
godep restore -v
