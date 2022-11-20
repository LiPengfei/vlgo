###
 # @Date: 2022-11-21 01:23:38
 # @LastEditors: lipengfei
 # @LastEditTime: 2022-11-21 01:45:40
 # @FilePath: \vlgo\proto\make_proto.sh
 # @Description: 
### 

plugin=gogofaster
tool=protoc-gen-gogofaster
version=v1.3.2
protos=(inner_code.proto error_code.proto game.proto)
bindir=../bin

function install_tool() {
    if [ ! -f $bindir/$tool ]; then
        go install github.com/gogo/protobuf/${tool}@$version
        cp -f $GOPATH/bin/$tool $bindir
    fi
}

function main() {
    install_tool
    mkdir -p pb_gen
    local depsDir=.:$GOPATH/pkg/mod/github.com/gogo/protobuf@$version

    for p in ${protos[@]}; do
        protoc -I=$depsDir --plugin=${tool}=$bindir/$tool--${plugin}_out=pb_gen $p 
    done
}

main