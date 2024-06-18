.PHONY: help proto grpcui run

help:
	@echo "Please use \`make <target>' where <target> is one of"
	@echo "  help            for this message"
	@echo "  swagger         to generate swagger docs"

swagger:
	swag init -g main.go --parseDependency --parseDepth 1 --parseInternal
