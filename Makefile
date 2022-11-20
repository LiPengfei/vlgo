version=0.0.1

GOOS=linux
GOARCH=amd64
VER=1.19.3
GO_VER=go${VER}

#GO=gotip
GO=GOOS=${GOOS} GOARCH=${GOARCH} ${GO_VER}

debug_flags= -x -v -mod=mod -race -gcflags="all=-N -l"
release_flags= -mod=mod -gcflags="all=-l"
compile_info_flags= -ldflags "-X 'buildinfo.Version=${version}'"
outdir=bin
cmddir=apps/cmd

bot-debug:
	go generate ./...
	${GO} build ${debug_flags} ${compile_info_flags} -o ${outdir}/bot ${cmddir}/bot/main.go

bot-release:
	go generate ./...
	${GO} build ${release_flags} ${compile_info_flags} -o ${outdir}/bot-release ${cmddir}/bot/main.go

install_lint:
	${GO} install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

lint:
	@which golangci-lint || make install_lint
	golangci-lint run --fix ./...

.PHONY: bot-release bot-debug lint
