package main

/*
	- https://github.com/hashicorp/terraform/releases/tag/v0.11.10
	- https://github.com/vishen/go-chromecast/releases/download/v0.0.3/go-chromecast_0.0.3_Linux_x86_64.tar.gz
	- https://github.com/protocolbuffers/protobuf/releases/download/v3.6.1/protoc-3.6.1-linux-x86_64.zip
	- https://dl.google.com/go/go1.11.2.linux-amd64.tar.gz
*/

var (
	terraform = Recipe{
		Name:    "terraform",
		URL:     "https://github.com/hashicorp/terraform/releases/tag/v{{ .Recipe.Version }}",
		Version: "0.11.10",
	}

	protoc = Recipe{
		Name:    "protoc",
		URL:     "https://github.com/protocolbuffers/protobuf/releases/download/v{{ .Recipe.Version }}/protoc-{{ .Recipe.Version }}-{{ .Config.OS }}-{{ .Config.Arch }}.zip",
		Version: "3.6.1",
	}
)
