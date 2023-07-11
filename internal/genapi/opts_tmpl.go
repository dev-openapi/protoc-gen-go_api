package genapi

import (
	"bytes"
	"html/template"
	"log"
)

var optsCode = `package {{ .GoPackage }}

import (
	"net/http"
	"errors"
)

type Option func(*Options)

type FnRequest func(*http.Client,*http.Request) (*http.Response, error)
type FnResponse func(*http.Response, interface{}) error

var (
	ErrNil = errors.New("resp nil")
	ErrNot200 = errors.New("resp not 200")
)

type Options struct {
	// do request
	DoRequest FnRequest
	// do response
	DoResponse FnResponse
	// addr
	addr string
	// client
	client *http.Client
}

func newOptions(opts ...Option) *Options {
	opt := Options{
		client: http.DefaultClient,
		DoRequest: doRequest,
		DoResponse: doResponse,
	}
	for _, o := range opts {
		o(&opt)
	}
	return &opt
}

func buildOptions(opt *Options, opts ...Option) *Options {
	res := newOptions(opts...)
	if len(res.addr) <= 0 {
		res.addr = opt.addr
	}
	return res
}

func doRequest(client *http.Client, req *http.Request) (*http.Response, error) {
	return client.Do(req)
}

func doResponse(resp *http.Response, a interface{}) error {
	if resp == nil {
		return ErrNil
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return ErrNot200
	}
	defer func(){_ = resp.Body.Close()}()
	bs, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	return json.Unmarshal(bs, a)
}

func WithDoRequest(fn FnRequest) Option {
	return func(o *Options) {
		o.DoRequest = fn
	}
}

func WithDoResponse(fn FnResponse) Option {
	return func(o *Options) {
		o.DoResponse = fn
	}
}

func WithClient(c *http.Client) Option {
	return func(o *Options) {
		o.client = c
	}
}

// addr must start with https:// or http://
func WithAddr(addr string) Option {
	return func(o *Options) {
		o.addr = addr
	}
}

`

func buildOptionsCode(data *OptionData) (string, error) {
	ocm, err := template.New("option_code_tmpl").Funcs(fn).Parse(optsCode)
	if err != nil {
		log.Println("parse option code template err: ", err)
		return "", err
	}
	bs := new(bytes.Buffer)
	err = ocm.Execute(bs, data)
	if err != nil {
		log.Println("execuete option code template err: ", err)
		return "", err
	}
	return bs.String(), nil
}
