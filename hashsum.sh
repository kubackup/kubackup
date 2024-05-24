#!/bin/bash
set -e

for file in dist/*
do
if [ -f "$file" ]
then
  if [[ "$OSTYPE" =~ ^linux ]]; then
  	sha256sum $file > $file.sha256
  elif [[ "$OSTYPE" =~ ^darwin ]]; then
  	shasum -a 256 $file > $file.sha256
  else
  	echo "Unsupported OS: $OSTYPE"
  	exit 1
  fi
fi
done


