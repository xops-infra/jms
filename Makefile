.PHONY: help proto grpcui run ssh-test

help:
	@echo "Please use \`make <target>' where <target> is one of"
	@echo "  help            for this message"
	@echo "  swagger         to generate swagger docs"
	@echo "  ssh-test        to connect to test env via ssh"

swagger:
	swag init -g main.go --parseDependency --parseDepth 1 --parseInternal

api:
	go run main.go api

ssh-test:
	ssh -p 22222 zhoushoujian@localhost
