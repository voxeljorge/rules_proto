package bufbuild

import (
	"fmt"
	"path"
	"strings"

	"github.com/bazelbuild/bazel-gazelle/label"
	"github.com/emicklei/proto"
	"github.com/stackb/rules_proto/pkg/plugin/golang/protobuf"
	"github.com/stackb/rules_proto/pkg/protoc"
)

func init() {
	protoc.Plugins().MustRegisterPlugin(&ConnectGoProto{})
}

// ConnectGoProto implements Plugin for the bufbuild/connect-go plugin.
type ConnectGoProto struct{}

// Name implements part of the Plugin interface.
func (p *ConnectGoProto) Name() string {
	return "bufbuild:connect-go"
}

func (p *ConnectGoProto) shouldApply(lib protoc.ProtoLibrary) bool {
	for _, f := range lib.Files() {
		if f.HasServices() {
			return true
		}
	}
	return false
}

// Configure implements part of the Plugin interface.
func (p *ConnectGoProto) Configure(ctx *protoc.PluginContext) *protoc.PluginConfiguration {
	if !p.shouldApply(ctx.ProtoLibrary) {
		return nil
	}

	options := ctx.PluginConfig.GetOptions()
	mappings, _ := protobuf.GetImportMappings(options)

	for k, v := range mappings {
		// "option" is used as the name since we cannot leave that part of the
		// label empty.
		fmt.Println("protoc-gen-connect-go-mapping:"+k+"="+v)
		protoc.GlobalResolver().Provide("proto", "M", k, label.New("", v, "option")) // FIXME(pcj): should this not be config.RepoName?
	}

	pc := &protoc.PluginConfiguration{
		Label:   label.New("build_stack_rules_proto", "plugin/bufbuild", "connect-go"),
		Outputs: p.outputs(ctx.ProtoLibrary, mappings),
		Options: options,
	}
	if len(pc.Outputs) == 0 {
		pc.Outputs = nil
	}
	return pc
}

func (p *ConnectGoProto) outputs(lib protoc.ProtoLibrary, importMappings map[string]string) []string {
	srcs := make([]string, 0)
	for _, f := range lib.Files() {
		if !f.HasServices() {
			continue
		}

		pkgName, ok := goPackageName(f)
		if !ok {
			continue
		}

		pbPath := protobuf.GetGoOutputBaseName(f, importMappings)
		pbDir := path.Dir(pbPath)
		pbName := path.Base(pbPath)
		outpath := path.Join(pbDir, pkgName+"connect", pbName+".connect.go")

		srcs = append(srcs, outpath)
	}
	return srcs
}

func goPackageName(f *protoc.File) (string, bool) {
	opt, ok := goPackageOption(f.Options())
	if !ok {
		return "", false
	}

	var vals []string
	if strings.Contains(opt, ";") {
		vals = strings.Split(opt, ";")
	} else {
		vals = strings.Split(opt, "/")
	}
	return vals[len(vals)-1], true
}

// goPackageOption is a utility function to seek for the go_package option.
func goPackageOption(options []proto.Option) (string, bool) {
	for _, opt := range options {
		if opt.Name != "go_package" {
			continue
		}
		return opt.Constant.Source, true
	}

	return "", false
}