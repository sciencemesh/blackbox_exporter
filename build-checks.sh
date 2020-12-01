#!/bin/bash
echo "Building BBE checks..."
cd ./checks
for dir in ./*/
do
    dir=${dir%*/}
    name="check_${dir:2}"
    echo " " $dir "->" $name
    go build -o "$name" "$dir"
done
cd ..
