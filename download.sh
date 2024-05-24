#!/bin/bash
set -e
version=$1
if [ -d "pkg/restic_source" ]; then
	rm -rf pkg/restic_source
fi

mkdir -p pkg/restic_source
wget -O restic.tar.gz https://github.com/restic/restic/archive/refs/tags/v"${version}".tar.gz
tar -zxvf restic.tar.gz
cp -rp restic-"${version}"/internal pkg/restic_source/rinternal
cp -rp restic-"${version}"/LICENSE pkg/restic_source/
cp -rp restic-"${version}"/VERSION pkg/restic_source/
rm -rf restic.tar.gz
rm -rf restic-"${version}"

if [[ "$OSTYPE" =~ ^linux ]]; then
	# linux
	# shellcheck disable=SC2046
	sed -i "s/\"github.com\/restic\/restic\/internal/\"github.com\/kubackup\/kubackup\/pkg\/restic_source\/rinternal/g" $(grep -rl "\"github.com\/restic\/restic\/internal" pkg/restic_source/rinternal)
elif [[ "$OSTYPE" =~ ^darwin ]]; then
	# darwin
	# shellcheck disable=SC2046
	sed -i '' "s/\"github.com\/restic\/restic\/internal/\"github.com\/kubackup\/kubackup\/pkg\/restic_source\/rinternal/g" $(grep -rl "\"github.com\/restic\/restic\/internal" pkg/restic_source/rinternal)
else
	echo "Unsupported OS: $OSTYPE"
	exit 1
fi
