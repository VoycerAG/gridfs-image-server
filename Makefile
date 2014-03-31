all: 
	go test -coverprofile fmt ...VoycerAG/config
	go test -coverprofile fmt ...VoycerAG/server

coverage-server:
	go test -coverprofile bin/coverage.out  .../VoycerAG/server
	go tool cover -html=bin/coverage.out

coverage-config:
	go test -coverprofile bin/coverage.out  .../VoycerAG/config
	go tool cover -html=bin/coverage.out