#!/bin/bash
set -e

restic_version="0.13.1"

go mod download
./download.sh ${restic_version}
go mod tidy

cd web/dashboard
npm install
