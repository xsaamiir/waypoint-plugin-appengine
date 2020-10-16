PLUGIN_NAME=gae

all: protos build

protos:
	@echo ""
	@echo "Build Protos"

	protoc -I . --go_opt=paths=source_relative --go_out=. ./builder/output.proto ./registry/output.proto ./platform/output.proto ./release/output.proto

build:
	@echo ""
	@echo "Compile Plugin"

	go build -o ./bin/waypoint-plugin-${PLUGIN_NAME} ./main.go 

install:
	@echo ""
	@echo "Installing Plugin"

	cp ./bin/waypoint-plugin-${PLUGIN_NAME} ${HOME}/.config/waypoint/plugins/   