all: 
	go test -coverprofile /dev/null ...VoycerAG/server

coverage:
	go test -coverprofile bin/coverage.out  .../VoycerAG/server
	go tool cover -html=bin/coverage.out

