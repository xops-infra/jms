.DEFAULT_GOAL := help

.PHONY: help swagger all api api-local ssh-test web

help:
	@echo "Please use \`make <target>' where <target> is one of"
	@echo "  help            for this message"
	@echo "  swagger         to generate swagger docs"
	@echo "  all             to rebuild and redeploy all jms services via docker compose"
	@echo "  api             to build and start jms-api via docker compose"
	@echo "  api-local       to run jms api locally with go run"
	@echo "  ssh-test        to connect to test env via ssh"
	@echo "  web             to build and start jms-web via docker compose"

swagger:
	swag init -g main.go --parseDependency --parseDepth 1 --parseInternal

all:
	docker compose build jms-sshd jms-scheduler jms-api jms-web
	docker compose up -d --force-recreate jms-sshd jms-scheduler jms-api jms-web

api-local:
	go run main.go api

api:
	docker compose build jms-api
	docker compose up -d jms-api

ssh-test:
	ssh -p 22222 zhoushoujian@localhost

web:
	docker compose build jms-web
	docker compose up -d jms-web
