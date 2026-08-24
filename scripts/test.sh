#!/usr/bin/env bash
set -euo pipefail

go install gotest.tools/gotestsum@latest
gotestsum --format testname -- -race -coverprofile=cover.out $(go list ./...)
