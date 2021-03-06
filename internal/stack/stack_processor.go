package stack

import (
	c "github.com/cloudposse/terraform-provider-utils/internal/convert"
	m "github.com/cloudposse/terraform-provider-utils/internal/merge"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"path"
)

// ProcessYAMLConfigFiles takes a list of paths to YAML config files, processes and deep-merges all imports,
// and returns a list of stack configs
func ProcessYAMLConfigFiles(filePaths []string) ([]string, error) {
	var result []string

	for _, p := range filePaths {
		config, err := ProcessYAMLConfigFile(p)
		if err != nil {
			return nil, err
		}

		finalConfig, err := ProcessConfig(p, config)
		if err != nil {
			return nil, err
		}

		yamlConfig, err := yaml.Marshal(finalConfig)
		if err != nil {
			return nil, err
		}

		result = append(result, string(yamlConfig))
	}

	return result, nil
}

// ProcessYAMLConfigFile takes a path to a YAML config file,
// recursively processes and deep-merges all imports,
// and returns stack config as map[interface{}]interface{}
func ProcessYAMLConfigFile(filePath string) (map[interface{}]interface{}, error) {
	var configs []map[interface{}]interface{}
	dir := path.Dir(filePath)

	stackYamlConfig, err := ioutil.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	stackMapConfig, err := c.YAMLToMapOfInterfaces(string(stackYamlConfig))
	if err != nil {
		return nil, err
	}

	// Find and process all imports
	if imports, ok := stackMapConfig["import"]; ok {
		for _, i := range imports.([]interface{}) {
			p := path.Join(dir, i.(string)+".yaml")

			yamlConfig, err := ProcessYAMLConfigFile(p)
			if err != nil {
				return nil, err
			}
			configs = append(configs, yamlConfig)
		}
	}

	configs = append(configs, stackMapConfig)

	// Deep-merge the config file and the imports
	result, err := m.Merge(configs)
	if err != nil {
		return nil, err
	}

	return result, nil
}

// ProcessConfig takes a raw stack config, deep-merges all variables and backends,
// and returns the final stack configuration for all Terraform and helmfile components
func ProcessConfig(stack string, config map[interface{}]interface{}) (map[interface{}]interface{}, error) {
	globalVarsSection := map[interface{}]interface{}{}
	globalSettingsSection := map[interface{}]interface{}{}
	globalEnvSection := map[interface{}]interface{}{}
	globalTerraformSection := map[interface{}]interface{}{}
	globalHelmfileSection := map[interface{}]interface{}{}
	globalComponentsSection := map[interface{}]interface{}{}

	terraformVars := map[interface{}]interface{}{}
	terraformSettings := map[interface{}]interface{}{}
	terraformEnv := map[interface{}]interface{}{}

	helmfileVars := map[interface{}]interface{}{}
	helmfileSettings := map[interface{}]interface{}{}
	helmfileEnv := map[interface{}]interface{}{}

	backendType := "s3"
	backend := map[interface{}]interface{}{}

	terraformComponents := map[string]interface{}{}
	helmfileComponents := map[string]interface{}{}
	allComponents := map[string]interface{}{}

	// Global sections
	if i, ok := config["vars"]; ok {
		globalVarsSection = i.(map[interface{}]interface{})
	}

	if i, ok := config["settings"]; ok {
		globalSettingsSection = i.(map[interface{}]interface{})
	}

	if i, ok := config["env"]; ok {
		globalEnvSection = i.(map[interface{}]interface{})
	}

	if i, ok := config["terraform"]; ok {
		globalTerraformSection = i.(map[interface{}]interface{})
	}

	if i, ok := config["helmfile"]; ok {
		globalHelmfileSection = i.(map[interface{}]interface{})
	}

	if i, ok := config["components"]; ok {
		globalComponentsSection = i.(map[interface{}]interface{})
	}

	// Terraform section
	if i, ok := globalTerraformSection["vars"]; ok {
		terraformVars = i.(map[interface{}]interface{})
	}

	globalAndTerraformVars, err := m.Merge([]map[interface{}]interface{}{globalVarsSection, terraformVars})
	if err != nil {
		return nil, err
	}

	if i, ok := globalTerraformSection["settings"]; ok {
		terraformSettings = i.(map[interface{}]interface{})
	}

	globalAndTerraformSettings, err := m.Merge([]map[interface{}]interface{}{globalSettingsSection, terraformSettings})
	if err != nil {
		return nil, err
	}

	if i, ok := globalTerraformSection["env"]; ok {
		terraformEnv = i.(map[interface{}]interface{})
	}

	globalAndTerraformEnv, err := m.Merge([]map[interface{}]interface{}{globalEnvSection, terraformEnv})
	if err != nil {
		return nil, err
	}

	if i, ok := globalTerraformSection["backend_type"]; ok {
		backendType = i.(string)
	}

	if i, ok := globalTerraformSection["backend"]; ok {
		if backendSection, backendSectionExist := i.(map[interface{}]interface{})[backendType]; backendSectionExist {
			backend = backendSection.(map[interface{}]interface{})
		}
	}

	// Helmfile section
	if i, ok := globalHelmfileSection["vars"]; ok {
		helmfileVars = i.(map[interface{}]interface{})
	}

	globalAndHelmfileVars, err := m.Merge([]map[interface{}]interface{}{globalVarsSection, helmfileVars})
	if err != nil {
		return nil, err
	}

	if i, ok := globalHelmfileSection["settings"]; ok {
		helmfileSettings = i.(map[interface{}]interface{})
	}

	globalAndHelmfileSettings, err := m.Merge([]map[interface{}]interface{}{globalSettingsSection, helmfileSettings})
	if err != nil {
		return nil, err
	}

	if i, ok := globalHelmfileSection["env"]; ok {
		helmfileEnv = i.(map[interface{}]interface{})
	}

	globalAndHelmfileEnv, err := m.Merge([]map[interface{}]interface{}{globalEnvSection, helmfileEnv})
	if err != nil {
		return nil, err
	}

	// Process all Terraform components
	if allTerraformComponents, ok := globalComponentsSection["terraform"]; ok {
		allTerraformComponentsMap := allTerraformComponents.(map[interface{}]interface{})

		for component, v := range allTerraformComponentsMap {
			componentMap := v.(map[interface{}]interface{})

			componentVars := map[interface{}]interface{}{}
			if i, ok2 := componentMap["vars"]; ok2 {
				componentVars = i.(map[interface{}]interface{})
			}

			componentSettings := map[interface{}]interface{}{}
			if i, ok2 := componentMap["settings"]; ok2 {
				componentSettings = i.(map[interface{}]interface{})
			}

			componentEnv := map[interface{}]interface{}{}
			if i, ok2 := componentMap["env"]; ok2 {
				componentEnv = i.(map[interface{}]interface{})
			}

			componentBackend := map[interface{}]interface{}{}
			if i, ok2 := componentMap["backend"]; ok2 {
				componentBackend = i.(map[interface{}]interface{})[backendType].(map[interface{}]interface{})
			}

			baseComponentVars := map[interface{}]interface{}{}
			baseComponentSettings := map[interface{}]interface{}{}
			baseComponentEnv := map[interface{}]interface{}{}
			baseComponentBackend := map[interface{}]interface{}{}
			baseComponentName := ""

			if baseComponent, baseComponentExist := componentMap["component"]; baseComponentExist {
				baseComponentName = baseComponent.(string)

				if baseComponentSection, baseComponentSectionExist := allTerraformComponentsMap[baseComponentName]; baseComponentSectionExist {
					baseComponentMap := baseComponentSection.(map[interface{}]interface{})

					if baseComponentVarsSection, baseComponentVarsSectionExist := baseComponentMap["vars"]; baseComponentVarsSectionExist {
						baseComponentVars = baseComponentVarsSection.(map[interface{}]interface{})
					}

					if baseComponentSettingsSection, baseComponentSettingsSectionExist := baseComponentMap["settings"]; baseComponentSettingsSectionExist {
						baseComponentSettings = baseComponentSettingsSection.(map[interface{}]interface{})
					}

					if baseComponentEnvSection, baseComponentEnvSectionExist := baseComponentMap["env"]; baseComponentEnvSectionExist {
						baseComponentEnv = baseComponentEnvSection.(map[interface{}]interface{})
					}

					if baseComponentBackendSection, baseComponentBackendSectionExist := baseComponentMap["backend"]; baseComponentBackendSectionExist {
						if backendTypeSection, backendTypeSectionExist := baseComponentBackendSection.(map[interface{}]interface{})[backendType]; backendTypeSectionExist {
							baseComponentBackend = backendTypeSection.(map[interface{}]interface{})
						}
					}
				} else {
					return nil, errors.New("Terraform component '" + component.(string) + "' defines attribute 'component: " +
						baseComponentName + "', " + "but `" + baseComponentName + "' is not defined in the stack '" + stack + "'")
				}
			}

			finalComponentVars, err := m.Merge([]map[interface{}]interface{}{globalAndTerraformVars, baseComponentVars, componentVars})
			if err != nil {
				return nil, err
			}

			finalComponentSettings, err := m.Merge([]map[interface{}]interface{}{globalAndTerraformSettings, baseComponentSettings, componentSettings})
			if err != nil {
				return nil, err
			}

			finalComponentEnv, err := m.Merge([]map[interface{}]interface{}{globalAndTerraformEnv, baseComponentEnv, componentEnv})
			if err != nil {
				return nil, err
			}

			finalComponentBackend, err := m.Merge([]map[interface{}]interface{}{backend, baseComponentBackend, componentBackend})
			if err != nil {
				return nil, err
			}

			comp := map[string]interface{}{}
			comp["vars"] = finalComponentVars
			comp["settings"] = finalComponentSettings
			comp["env"] = finalComponentEnv
			comp["backend_type"] = backendType
			comp["backend"] = finalComponentBackend

			if baseComponentName != "" {
				comp["component"] = baseComponentName
			}

			terraformComponents[component.(string)] = comp
		}
	}

	// Process all helmfile components
	if allHelmfileComponents, ok := globalComponentsSection["helmfile"]; ok {
		allHelmfileComponentsMap := allHelmfileComponents.(map[interface{}]interface{})

		for component, v := range allHelmfileComponentsMap {
			componentMap := v.(map[interface{}]interface{})

			componentVars := map[interface{}]interface{}{}
			if i2, ok2 := componentMap["vars"]; ok2 {
				componentVars = i2.(map[interface{}]interface{})
			}

			componentSettings := map[interface{}]interface{}{}
			if i, ok2 := componentMap["settings"]; ok2 {
				componentSettings = i.(map[interface{}]interface{})
			}

			componentEnv := map[interface{}]interface{}{}
			if i, ok2 := componentMap["env"]; ok2 {
				componentEnv = i.(map[interface{}]interface{})
			}

			finalComponentVars, err := m.Merge([]map[interface{}]interface{}{globalAndHelmfileVars, componentVars})
			if err != nil {
				return nil, err
			}

			finalComponentSettings, err := m.Merge([]map[interface{}]interface{}{globalAndHelmfileSettings, componentSettings})
			if err != nil {
				return nil, err
			}

			finalComponentEnv, err := m.Merge([]map[interface{}]interface{}{globalAndHelmfileEnv, componentEnv})
			if err != nil {
				return nil, err
			}

			comp := map[string]interface{}{}
			comp["vars"] = finalComponentVars
			comp["settings"] = finalComponentSettings
			comp["env"] = finalComponentEnv
			helmfileComponents[component.(string)] = comp
		}
	}

	allComponents["terraform"] = terraformComponents
	allComponents["helmfile"] = helmfileComponents

	result := map[interface{}]interface{}{
		"components": allComponents,
	}

	return result, nil
}
