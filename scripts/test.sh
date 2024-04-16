#!/usr/bin/env bash

go install gotest.tools/gotestsum@latest
gotestsum --format testname -- -race -coverprofile=cover.out $(go list ./...)
