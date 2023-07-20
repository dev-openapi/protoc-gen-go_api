package genapi

import (
	"regexp"

	"github.com/golang/protobuf/protoc-gen-go/descriptor"
)

type restInfo struct {
	verb, route, body, typ string
}

const (
	BODY_JSON  = "json"
	BODY_FORM  = "form"
	BODY_MULTI = "multi"
	BODY_BYTE  = "byte"
)

const (
	emptyValue = "google.protobuf.Empty"
	// protoc puts a dot in front of name, signaling that the name is fully qualified.
	emptyType               = "." + emptyValue
	lroType                 = ".google.longrunning.Operation"
	httpBodyType            = ".google.api.HttpBody"
	alpha                   = "alpha"
	beta                    = "beta"
	disableDeadlinesVar     = "GOOGLE_API_GO_EXPERIMENTAL_DISABLE_DEFAULT_DEADLINE"
	fieldTypeBool           = descriptor.FieldDescriptorProto_TYPE_BOOL
	fieldTypeString         = descriptor.FieldDescriptorProto_TYPE_STRING
	fieldTypeBytes          = descriptor.FieldDescriptorProto_TYPE_BYTES
	fieldTypeMessage        = descriptor.FieldDescriptorProto_TYPE_MESSAGE
	fieldLabelRepeated      = descriptor.FieldDescriptorProto_LABEL_REPEATED
	defaultPollInitialDelay = "time.Second" // 1 second
	defaultPollMaxDelay     = "time.Minute" // 1 minute
)

var wellKnownTypes = []string{
	".google.protobuf.FieldMask",
	".google.protobuf.Timestamp",
	".google.protobuf.Duration",
	".google.protobuf.DoubleValue",
	".google.protobuf.FloatValue",
	".google.protobuf.Int64Value",
	".google.protobuf.UInt64Value",
	".google.protobuf.Int32Value",
	".google.protobuf.UInt32Value",
	".google.protobuf.BoolValue",
	".google.protobuf.StringValue",
	".google.protobuf.BytesValue",
	".google.protobuf.Value",
	".google.protobuf.ListValue",
}

var httpPatternVarRegex = regexp.MustCompile(`{([a-zA-Z0-9_.]+?)(=[^{}]+)?}`)
