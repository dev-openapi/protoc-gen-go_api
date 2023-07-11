package genapi

import (
	"fmt"
	"path"
	"strings"

	"github.com/golang/protobuf/protoc-gen-go/descriptor"
	plugin "github.com/golang/protobuf/protoc-gen-go/plugin"
	"google.golang.org/protobuf/proto"
)

func Gen(req *plugin.CodeGeneratorRequest) (*plugin.CodeGeneratorResponse, error) {
	initDescInfo(req)
	initComment(req)
	opts, err := parseOptions(req.Parameter)
	if err != nil {
		return nil, err
	}
	var resp plugin.CodeGeneratorResponse
	optdata := &OptionData{}
	for _, f := range req.GetProtoFile() {
		if !strContains(req.GetFileToGenerate(), f.GetName()) {
			continue
		}
		data, err := parseRestFile(f)
		if err != nil {
			return nil, err
		}
		if len(optdata.GoPackage) == 0 {
			optdata.GoPackage = data.GoPackage
		}
		bs, err := buildFrame(data)
		if err != nil {
			return nil, err
		}
		name := fmt.Sprintf("%s.api.go", strings.ReplaceAll(f.GetName(), ".proto", ""))
		if len(opts.out) > 0 {
			name = path.Join(opts.out, name)
		}
		resp.File = append(resp.File, &plugin.CodeGeneratorResponse_File{
			Name:    proto.String(name),
			Content: proto.String(bs),
		})
	}
	bs, err := buildOptionsCode(optdata)
	if err != nil {
		return nil, err
	}
	resp.File = append(resp.File, &plugin.CodeGeneratorResponse_File{
		Name:    proto.String("option.go"),
		Content: proto.String(bs),
	})
	return &resp, nil
}

func parseRestFile(fd *descriptor.FileDescriptorProto) (*FileData, error) {
	pkg := fd.Options.GetGoPackage()
	ps := strings.Split(pkg, "/")
	data := &FileData{
		Version:   Version,
		Source:    fd.GetName(),
		GoPackage: strings.ReplaceAll(ps[len(ps)-1], "-", "_"),
	}
	servs := fd.GetService()

	for _, serv := range servs {
		srv, err := parseRestService(fd, serv)
		if err != nil {
			return nil, err
		}
		data.Services = append(data.Services, srv)
	}

	return data, nil
}

func parseRestService(fd *descriptor.FileDescriptorProto, serv *descriptor.ServiceDescriptorProto) (*ServiceData, error) {
	data := &ServiceData{
		PkgName:  fd.GetPackage(),
		ServName: strings.ReplaceAll(serv.GetName(), "Service", ""),
	}

	meths := serv.GetMethod()
	for _, meth := range meths {
		mth, err := parseRestMethod(fd, serv, meth)
		if err != nil {
			return nil, err
		}
		data.Methods = append(data.Methods, mth)
	}

	return data, nil
}

func parseRestMethod(fd *descriptor.FileDescriptorProto, serv *descriptor.ServiceDescriptorProto, meth *descriptor.MethodDescriptorProto) (*MethodData, error) {
	data := &MethodData{
		ServName: strings.ReplaceAll(serv.GetName(), "Service", ""),
		MethName: meth.GetName(),
		Comment:  getComment(meth),
		ReqTyp:   typeName(meth.GetInputType()),
		ResTyp:   typeName(meth.GetOutputType()),
	}
	switch {
	case meth.GetClientStreaming():
		data.ReqCode = fmt.Sprintf(noClientStream, meth.GetName())
	case meth.GetServerStreaming():
		data.ReqCode = fmt.Sprintf(noServerStream, meth.GetName())
	default:
		code, err := genRestMethodCode(fd, serv, meth)
		if err != nil {
			return nil, err
		}
		data.ReqCode = code
	}

	return data, nil
}
