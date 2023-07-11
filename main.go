package main

import (
	"io/ioutil"
	"log"
	"os"

	"github.com/dev-openapi/protoc-gen-go_api/internal/genapi"
	"google.golang.org/protobuf/proto"

	plugin "github.com/golang/protobuf/protoc-gen-go/plugin"
)

func main() {
	reqBytes, err := ioutil.ReadAll(os.Stdin)
	if err != nil {
		log.Fatal(err)
		return
	}
	var genReq plugin.CodeGeneratorRequest
	if err := proto.Unmarshal(reqBytes, &genReq); err != nil {
		log.Fatal(err)
	}

	genResp, err := genapi.Gen(&genReq)
	if err != nil {
		genResp.Error = proto.String(err.Error())
	}

	genResp.SupportedFeatures = proto.Uint64(uint64(plugin.CodeGeneratorResponse_FEATURE_PROTO3_OPTIONAL))

	outBytes, err := proto.Marshal(genResp)
	if err != nil {
		log.Fatal(err)
	}
	if _, err := os.Stdout.Write(outBytes); err != nil {
		log.Fatal(err)
	}
}
