all:
	GOPATH=`pwd` go test -v github.com/VoycerAG/config

project:
	GOPATH=`pwd` go install github.com/VoycerAG
