package genapi

import (
	"fmt"
	"net/http"
	"regexp"
	"sort"
	"strings"

	"github.com/dev-openapi/protoc-gen-go_api/internal/pbinfo"
	"github.com/golang/protobuf/protoc-gen-go/descriptor"
	plugin "github.com/golang/protobuf/protoc-gen-go/plugin"
	"google.golang.org/genproto/googleapis/api/annotations"
	"google.golang.org/protobuf/proto"
)

var descInfo pbinfo.Info

func initDescInfo(req *plugin.CodeGeneratorRequest) {
	descInfo = pbinfo.Of(req.GetProtoFile())
}

func genRestMethodCode(fd *descriptor.FileDescriptorProto, serv *descriptor.ServiceDescriptorProto, meth *descriptor.MethodDescriptorProto) (string, error) {
	rest := buildRestInfo(meth)
	if rest == nil {
		return fmt.Sprintf(noRestyOptions, meth.GetName()), nil
	}

	data := &CodeData{}
	data.Verb = strings.ToUpper(rest.verb)

	tokens := buildRoute(rest)
	route := rest.route
	// TODO(noahdietz): handle more complex path urls involving = and *,
	// e.g. v1beta1/repeat/{info.f_string=first/*}/{info.f_child.f_string=second/**}:pathtrailingresource
	route = httpPatternVarRegex.ReplaceAllStringFunc(route, func(s string) string { return "%v" })
	route = fmt.Sprintf("%s%s", "%s", route)
	if len(tokens) > 0 {
		data.RouteCode = fmt.Sprintf("rawURL := fmt.Sprintf(%q, opt.addr, %s)\n", route, strings.Join(tokens, ","))
	} else {
		data.RouteCode = fmt.Sprintf("rawURL := fmt.Sprintf(%q, opt.addr)\n", route)
	}

	data.BodyCode = buildBody(meth, rest)

	query := buildQuery(meth)
	data.QueryCode = strings.Join(query, "\n\t")

	return buildRequestCode(data)
}

func buildBody(m *descriptor.MethodDescriptorProto, rest *restInfo) string {
	body, typ := "nil", BODY_JSON
	if rest.body != "" {
		typ = rest.typ
		body = "in"
		if rest.body != "*" {
			body = fmt.Sprintf("in%s", fieldGetter(rest.body))
		}
	}
	if body == "nil" {
		return ""
	}
	code := strings.Builder{}
	var bc string
	switch typ {
	case BODY_FORM:
		forms := buildBodyForm(m, rest)
		if len(forms) <= 0 {
			break
		}
		bc, _ = buildBodyFormCode(strings.Join(forms, "\n\t"))
	case BODY_MULTI:
		forms := buildBodyForm(m, rest)
		if len(forms) <= 0 {
			break
		}
		bc, _ = buildBodyMultiCode(strings.Join(forms, "\n\t"))
	default:
		bc, _ = buildBodyJsonCode(body)
	}
	code.WriteString(bc)
	return code.String()
}

func buildBodyForm(m *descriptor.MethodDescriptorProto, rest *restInfo) []string {
	queryParams := map[string]*descriptor.FieldDescriptorProto{}
	request := descInfo.Type[m.GetInputType()].(*descriptor.DescriptorProto)
	if rest.body != "*" {
		bodyField := lookupField(m.GetInputType(), rest.body)
		request = descInfo.Type[bodyField.GetTypeName()].(*descriptor.DescriptorProto)
	}

	// Possible query parameters are all leaf fields in the request or body.
	pathToLeaf := getLeafs(request, nil)
	// Iterate in sorted order to
	for path, leaf := range pathToLeaf {
		// If, and only if, a leaf field is not a path parameter or a body parameter,
		// it is a query parameter.
		if lookupField(request.GetName(), leaf.GetName()) == nil {
			queryParams[path] = leaf
		}
	}
	fmtKey := "bodyForms[%q] = %s"
	return formParams(fmtKey, queryParams)
}

func buildRoute(rest *restInfo) []string {
	tokens := []string{}
	// Can't just reuse pathParams because the order matters
	for _, path := range httpPatternVarRegex.FindAllStringSubmatch(rest.route, -1) {
		// In the returned slice, the zeroth element is the full regex match,
		// and the subsequent elements are the sub group matches.
		// See the docs for FindStringSubmatch for further details.
		tokens = append(tokens, fmt.Sprintf("in%s", fieldGetter(path[1])))
	}
	return tokens
}

func buildQuery(m *descriptor.MethodDescriptorProto) []string {
	params := buildParams(m)
	str := `params.Add("%s", %s)`
	return formParams(str, params)
}

func buildParams(m *descriptor.MethodDescriptorProto) map[string]*descriptor.FieldDescriptorProto {
	queryParams := map[string]*descriptor.FieldDescriptorProto{}
	info := buildRestInfo(m)
	if info == nil {
		return queryParams
	}
	if info.body == "*" {
		// The entire request is the REST body.
		return queryParams
	}

	pathParams := pathParams(m)
	// Minor hack: we want to make sure that the body parameter is NOT a query parameter.
	pathParams[info.body] = &descriptor.FieldDescriptorProto{}

	request := descInfo.Type[m.GetInputType()].(*descriptor.DescriptorProto)
	// Body parameters are fields present in the request body.
	// This may be the request message itself or a subfield.
	// Body parameters are not valid query parameters,
	// because that means the same param would be sent more than once.
	bodyField := lookupField(m.GetInputType(), info.body)

	// Possible query parameters are all leaf fields in the request or body.
	pathToLeaf := getLeafs(request, bodyField)
	// Iterate in sorted order to
	for path, leaf := range pathToLeaf {
		// If, and only if, a leaf field is not a path parameter or a body parameter,
		// it is a query parameter.
		if _, ok := pathParams[path]; !ok && lookupField(request.GetName(), leaf.GetName()) == nil {
			queryParams[path] = leaf
		}
	}

	return queryParams
}

func buildRestInfo(m *descriptor.MethodDescriptorProto) *restInfo {
	if m == nil || m.GetOptions() == nil {
		return nil
	}
	anno := proto.GetExtension(m.GetOptions(), annotations.E_Http)

	rule := anno.(*annotations.HttpRule)
	info := restInfo{}
	body := rule.GetBody()
	if len(body) == 0 {
		info.body = ""
	}
	bs := strings.Split(body, ",")
	if len(bs) == 1 {
		info.body = body
		info.typ = BODY_JSON
	} else {
		info.body = bs[0]
		info.typ = bs[1]
	}
	switch rule.GetPattern().(type) {
	case *annotations.HttpRule_Get:
		info.verb = http.MethodGet
		info.route = rule.GetGet()
	case *annotations.HttpRule_Post:
		info.verb = http.MethodPost
		info.route = rule.GetPost()
	case *annotations.HttpRule_Patch:
		info.verb = http.MethodPatch
		info.route = rule.GetPatch()
	case *annotations.HttpRule_Put:
		info.verb = http.MethodPut
		info.verb = rule.GetPut()
	}
	return &info
}

func pathParams(m *descriptor.MethodDescriptorProto) map[string]*descriptor.FieldDescriptorProto {
	pathParams := map[string]*descriptor.FieldDescriptorProto{}
	info := buildRestInfo(m)
	if info == nil {
		return pathParams
	}

	// Match using the curly braces but don't include them in the grouping.
	re := regexp.MustCompile("{([^}]+)}")
	for _, p := range re.FindAllStringSubmatch(info.route, -1) {
		// In the returned slice, the zeroth element is the full regex match,
		// and the subsequent elements are the sub group matches.
		// See the docs for FindStringSubmatch for further details.
		param := strings.Split(p[1], "=")[0]
		field := lookupField(m.GetInputType(), param)
		if field == nil {
			continue
		}
		pathParams[param] = field
	}

	return pathParams
}

func lookupField(msgName, field string) *descriptor.FieldDescriptorProto {
	var desc *descriptor.FieldDescriptorProto
	msg := descInfo.Type[msgName]

	// If the message doesn't exist, fail cleanly.
	if msg == nil {
		return desc
	}

	msgProto := msg.(*descriptor.DescriptorProto)
	msgFields := msgProto.GetField()

	// Split the key name for nested fields, and traverse the message chain.
	for _, seg := range strings.Split(field, ".") {
		// Look up the desired field by name, stopping if the leaf field is
		// found, continuing if the field is a nested message.
		for _, f := range msgFields {
			if f.GetName() == seg {
				desc = f

				// Search the nested message for the next segment of the
				// nested field chain.
				if f.GetType() == descriptor.FieldDescriptorProto_TYPE_MESSAGE {
					msg = descInfo.Type[f.GetTypeName()]
					msgProto = msg.(*descriptor.DescriptorProto)
					msgFields = msgProto.GetField()
				}
				break
			}
		}
	}
	return desc
}

func formParams(fmtKey string, queryParams map[string]*descriptor.FieldDescriptorProto) []string {
	// We want to iterate over fields in a deterministic order
	// to prevent spurious deltas when regenerating gapics.
	fields := make([]string, 0, len(queryParams))
	for p := range queryParams {
		fields = append(fields, p)
	}
	sort.Strings(fields)
	params := make([]string, 0, len(fields))

	for _, path := range fields {
		field := queryParams[path]
		required := isRequired(field)
		accessor := fieldGetter(path)
		singularPrimitive := field.GetType() != fieldTypeMessage &&
			field.GetType() != fieldTypeBytes &&
			field.GetLabel() != fieldLabelRepeated
		// key用命名的
		key := path

		var paramAdd string
		// Handle well known protobuf types with special JSON encodings.
		if strContains(wellKnownTypes, field.GetTypeName()) {
			b := strings.Builder{}
			b.WriteString(fmt.Sprintf("%s, err := json.Marshal(in%s)\n", field.GetJsonName(), accessor))
			b.WriteString("if err != nil {\n")
			b.WriteString("  return nil, err\n")
			b.WriteString("}\n")
			b.WriteString(fmt.Sprintf(fmtKey, key, fmt.Sprintf("string(%s)", field.GetJsonName())))
			paramAdd = b.String()
		} else {
			paramAdd = fmt.Sprintf(fmtKey, key, fmt.Sprintf("fmt.Sprintf(%q, in%s)", "%v", accessor))
		}

		// Only required, singular, primitive field types should be added regardless.
		if required && singularPrimitive {
			// Use string format specifier here in order to allow %v to be a raw string.
			params = append(params, paramAdd)
			continue
		}

		if field.GetLabel() == fieldLabelRepeated {
			// It's a slice, so check for len > 0, nil slice returns 0.
			params = append(params, fmt.Sprintf("if items := in%s; len(items) > 0 {", accessor))
			b := strings.Builder{}
			b.WriteString("for _, item := range items {\n")
			b.WriteString(fmt.Sprintf(fmtKey, key, fmt.Sprintf("fmt.Sprintf(%q, item)\n", "%v")))
			b.WriteString("}")
			paramAdd = b.String()

		} else if field.GetProto3Optional() {
			// Split right before the raw access
			toks := strings.Split(path, ".")
			toks = toks[:len(toks)-1]
			parentField := fieldGetter(strings.Join(toks, "."))
			directLeafField := directAccess(path)
			params = append(params, fmt.Sprintf("if in%s != nil && in%s != nil {", parentField, directLeafField))
		} else {
			// Default values are type specific
			switch field.GetType() {
			// Degenerate case, field should never be a message because that implies it's not a leaf.
			case fieldTypeMessage, fieldTypeBytes:
				params = append(params, fmt.Sprintf("if in%s != nil {", accessor))
			case fieldTypeString:
				params = append(params, fmt.Sprintf(`if in%s != "" {`, accessor))
			case fieldTypeBool:
				params = append(params, fmt.Sprintf(`if in%s {`, accessor))
			default: // Handles all numeric types including enums
				params = append(params, fmt.Sprintf(`if in%s != 0 {`, accessor))
			}
		}
		params = append(params, fmt.Sprintf("\t%s", paramAdd))
		params = append(params, "}")
	}

	return params
}

// Returns a map from fully qualified path to field descriptor for all the leaf fields of a message 'm',
// where a "leaf" field is a non-message whose top message ancestor is 'm'.
// e.g. for a message like the following
//
//	message Mollusc {
//	    message Squid {
//	        message Mantle {
//	            int32 mass_kg = 1;
//	        }
//	        Mantle mantle = 1;
//	    }
//	    Squid squid = 1;
//	}
//
// The one entry would be
// "squid.mantle.mass_kg": *descriptor.FieldDescriptorProto...
func getLeafs(msg *descriptor.DescriptorProto, excludedFields ...*descriptor.FieldDescriptorProto) map[string]*descriptor.FieldDescriptorProto {
	pathsToLeafs := map[string]*descriptor.FieldDescriptorProto{}

	contains := func(fields []*descriptor.FieldDescriptorProto, field *descriptor.FieldDescriptorProto) bool {
		for _, f := range fields {
			if field == f {
				return true
			}
		}
		return false
	}

	// We need to declare and define this function in two steps
	// so that we can use it recursively.
	var recurse func([]*descriptor.FieldDescriptorProto, *descriptor.DescriptorProto)

	handleLeaf := func(field *descriptor.FieldDescriptorProto, stack []*descriptor.FieldDescriptorProto) {
		elts := []string{}
		for _, f := range stack {
			elts = append(elts, f.GetName())
		}
		elts = append(elts, field.GetName())
		key := strings.Join(elts, ".")
		pathsToLeafs[key] = field
	}

	handleMsg := func(field *descriptor.FieldDescriptorProto, stack []*descriptor.FieldDescriptorProto) {
		if field.GetLabel() == descriptor.FieldDescriptorProto_LABEL_REPEATED {
			// Repeated message fields must not be mapped because no
			// client library can support such complicated mappings.
			// https://cloud.google.com/endpoints/docs/grpc-service-config/reference/rpc/google.api#grpc-transcoding
			return
		}
		if contains(excludedFields, field) {
			return
		}
		// Short circuit on infinite recursion
		if contains(stack, field) {
			return
		}

		subMsg := descInfo.Type[field.GetTypeName()].(*descriptor.DescriptorProto)
		recurse(append(stack, field), subMsg)
	}

	recurse = func(
		stack []*descriptor.FieldDescriptorProto,
		m *descriptor.DescriptorProto,
	) {
		for _, field := range m.GetField() {
			if field.GetType() == fieldTypeMessage && !strContains(wellKnownTypes, field.GetTypeName()) {
				handleMsg(field, stack)
			} else {
				handleLeaf(field, stack)
			}
		}
	}

	recurse([]*descriptor.FieldDescriptorProto{}, msg)
	return pathsToLeafs
}

// isRequired returns if a field is annotated as REQUIRED or not.
func isRequired(field *descriptor.FieldDescriptorProto) bool {
	if field.GetOptions() == nil {
		return false
	}

	eBehav := proto.GetExtension(field.GetOptions(), annotations.E_FieldBehavior)

	behaviors := eBehav.([]annotations.FieldBehavior)
	for _, b := range behaviors {
		if b == annotations.FieldBehavior_REQUIRED {
			return true
		}
	}

	return false
}
