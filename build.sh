#!/bin/bash

target="uct.go"
uctVersion="1.0.0"

platforms="darwin/386/macosx/i586 darwin/amd64/macosx/x64 linux/386/linux/i586 linux/amd64/linux/x64 windows/386/windows/i586 windows/amd64/windows/x64"

for platform in ${platforms}
do
    split=(${platform//\// })
    goos=${split[0]}
    goarch=${split[1]}
    pos=${split[2]}
    parch=${split[3]}

    echo "Building uct for $goos/$goarch platform..."

    # ensure output file name
    output="$binary"
    test "$output" || output="$(basename $target | sed 's/\.go//')"

    # add exe to windows output
    [[ "windows" == "$goos" ]] && output="$output.exe"

    zipfile="uct-$uctVersion-$pos-$parch"
    zipdir="$(dirname $target)/build/target/$zipfile"
    mkdir -p $zipdir
    # cp -r "$(dirname $target)/cmd" $zipdir
    cp -r "$(dirname $target)/res" $zipdir
    cp -r "$(dirname $target)/README.md" $zipdir
    cp -r "$(dirname $target)/LICENSE.txt" $zipdir

    # set destination path for binary
    destination="$zipdir/bin/$output"

    #echo "GOOS=$goos GOARCH=$goarch go build -x -o $destination $target"
    GOOS=$goos GOARCH=$goarch go build -ldflags "-X main.uctVersion=$uctVersion -X 'main.buildDate=$(date -u '+%Y-%m-%d %H:%M:%S')'" -o $destination $target

    pwd=`pwd`
    cd "$(dirname $target)/build/target"
    zip -r "$zipfile.zip" $zipfile > /dev/null 2>&1
    rm -rf $zipfile
    cd $pwd
done
