.PHONY: generate-server

generate-server:
	oapi-codegen -generate types,server,spec -package http -o internal/controller/http/api.go api/services.yaml

