#!/bin/bash
set -e

restic_version="0.16.5"

sh download.sh ${restic_version}
go mod download
go mod tidy
