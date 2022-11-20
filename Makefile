version=0.0.1

GOOS=linux
GOARCH=amd64
VER=1.19.3
GO_VER=go${VER}

#GO=gotip
GO=GOOS=${GOOS} GOARCH=${GOARCH} ${GO_VER}

debug_flags= -v -mod=mod -race -gcflags="all=-N -l"
release_flags= -mod=mod -gcflags="all=-l"
compile_info_flags= -ldflags "-X 'buildinfo.Version=${version}'"
outdir=bin
cmddir=apps/cmd/

install_lint:
	${GO} install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

lint: proto
	@which golangci-lint || make install_lint
	golangci-lint run --fix ./...

bot-release:
	${GO} build ${release_flags} ${compile_info_flags} -o ${outdir}/bot-release ${cmddir}/bot/main.go

bot-debug:
	${GO} build ${debug_flags} ${compile_info_flags} -o ${outdir}/bot ${cmddir}/bot/main.go

.PHONY: bot-release bot-debug