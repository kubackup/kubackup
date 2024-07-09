#!/bin/bash
set -e

restic_version="0.16.5"

go mod download
./download.sh ${restic_version}
go mod tidy

