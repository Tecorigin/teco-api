#!/bin/bash

echo "building web..."
cd web/default
bun run build
cd ../..
echo "build new-api..."
rm new-api
go build -o new-api
echo "done!!!"
md5sum new-api
