#!/bin/bash
set -e

restic_version="0.18.1"

sh download.sh ${restic_version}
go mod download
go mod tidy
