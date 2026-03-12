package cmd

import (
	"errors"
	"strings"

	"github.com/gruntwork-io/terragrunt/util"
	"github.com/hashicorp/terraform-config-inspect/tfconfig"
	"github.com/zclconf/go-cty/cty"
)

var localModuleSourcePrefixes = []string{
	"./",
	"../",
	".\\",
	"..\\",
}

func parseTerraformLocalModuleSource(path string) ([]string, error) {
	return parseTerraformLocalModuleSourceWithInputs(path, nil)
}

func parseTerraformLocalModuleSourceWithInputs(path string, inputs map[string]cty.Value) ([]string, error) {
	var module *tfconfig.Module
	var diags tfconfig.Diagnostics

	if inputs != nil && len(inputs) > 0 {
		module, diags = tfconfig.LoadModuleWithInputs(path, inputs)
	} else {
		module, diags = tfconfig.LoadModule(path)
	}

	if diags.HasErrors() {
		return nil, errors.New(diags.Error())
	}

	var sourceMap = map[string]bool{}
	for _, mc := range module.ModuleCalls {
		if isLocalTerraformModuleSource(mc.Source) {
			modulePath := util.JoinPath(path, mc.Source)
			modulePathGlob := util.JoinPath(modulePath, "*.tf*")

			if _, exists := sourceMap[modulePathGlob]; exists {
				continue
			}
			sourceMap[modulePathGlob] = true

			// Extract inputs from this module call to pass to the child module
			childInputs := make(map[string]cty.Value)

			if mc.Inputs != nil {
				for k, v := range mc.Inputs {
					// Pass all known, non-null values (strings, numbers, bools, objects, etc.)
					if v.IsKnown() && !v.IsNull() {
						childInputs[k] = v
					}
				}
			}

			// find local module source recursively with child inputs
			subSources, err := parseTerraformLocalModuleSourceWithInputs(modulePath, childInputs)
			if err != nil {
				return nil, err
			}

			for _, subSource := range subSources {
				sourceMap[subSource] = true
			}
		}
	}

	var sources = []string{}
	for source := range sourceMap {
		sources = append(sources, source)
	}

	return sources, nil
}

func isLocalTerraformModuleSource(raw string) bool {
	for _, prefix := range localModuleSourcePrefixes {
		if strings.HasPrefix(raw, prefix) {
			return true
		}
	}

	return false
}
