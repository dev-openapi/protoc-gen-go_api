# protoc-gen-go_api
一款用protobuf文件生成go的http调用代码。

## 安装

```bash
go install github.com/dev-openapi/protoc-gen-go_api@latest
```

## 使用

依赖googleapis的http.proto和annotations.proto。并且需要放到google/api/目录下

如果google/api是在工程文件目录下，执行以下脚本

```bash
protoc --go_api_out=out={dir}:. *.proto
```

如果google/api在其他文件目录下，执行以下脚本

```bash
protoc --proto_path={yourpath}:. --go_api_out=:. *.proto
```

## 注意

最新版本的protoc-gen-go要求go_package必须含有/，且会生成到$GOPATH/src目录下，所以建议把工程文件放到$GOPATH/src/git域名/git_group/目录下。

如 https://github.com/dev-openapi/wx-miniprogram 则该工程文件为 $GOPATH/src/github.com/dev-openapi/wx-miniprogram

另外，也可以指定protoc-gen-go的生成目录和设置source_relative相关配置，也可以生成到指定目录

如
```shell
protoc --go_out=.. --go_opt=paths=source_relative --go_api_out=out=..:. *.proto
```

这样会生成到上一层目录
