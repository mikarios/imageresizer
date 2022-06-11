// nolint:testpackage // access to internal functions needed
package config

import (
	"bufio"
	"embed"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/mikarios/golib/slices"

	"github.com/mikarios/imageresizer/internal/constants"
)

const (
	defaultTagName = "default"
)

var (
	//go:embed daemonConfigTemplates/*.env
	configs       embed.FS
	serversConfig = make(map[constants.ServerType]map[string]interface{})
	errInvalidEnv = errors.New("invalid environment")
)

func TestConfig(t *testing.T) {
	t.Parallel()

	for _, server := range constants.ServerTypeList {
		configName := fmt.Sprintf("daemonConfigTemplates/%s.env", server)

		configFile, err := configs.ReadFile(configName)
		if err != nil {
			panic(err)
		}

		serversConfig[server] = sanitizeConfigFile(t, configFile)
	}

	config := Config{}
	configType := reflect.TypeOf(config)

	checkTags(t, configType)

	for k, v := range serversConfig {
		if len(v) > 0 {
			t.Errorf(
				"there are some keys in %s.env which are not correctly set in config struct. Please add the correct servers [%v]",
				k,
				v,
			)
			t.Fail()
		}
	}
}

func sanitizeConfigFile(t *testing.T, configFile []byte) map[string]interface{} {
	t.Helper()

	keys := make(map[string]interface{}, 0)

	scanner := bufio.NewScanner(strings.NewReader(string(configFile)))
	for scanner.Scan() {
		txt := scanner.Text()
		if strings.HasPrefix(txt, "#") {
			continue
		}

		key := strings.Split(txt, "=")[0]
		if _, ok := keys[key]; ok {
			t.Errorf("found duplicate key %s", key)
			t.Fail()
		}

		if key != "" {
			keys[key] = struct{}{}
		}
	}

	return keys
}

func checkTags(t *testing.T, configType reflect.Type) {
	t.Helper()

	for i := 0; i < configType.NumField(); i++ {
		field := configType.Field(i)

		if field.Type.Kind() == reflect.Struct {
			checkTags(t, field.Type)
			continue
		}

		if err := validateTags(&field); err != nil {
			t.Errorf("%v %s", field.Name, err)
			t.Fail()
		}
	}
}

func validateTags(field *reflect.StructField) error {
	serversStr := field.Tag.Get(serversTagName)

	servers := strings.Split(serversStr, ",")
	if len(servers) == 0 {
		return fmt.Errorf("%w: missing %s field", errInvalidEnv, serversTagName)
	}

	for _, server := range servers {
		if !slices.Contains(constants.ServerTypeList, constants.ServerType(server)) {
			return fmt.Errorf("%w: unknown server name %s", errInvalidEnv, server)
		}

		envConfigTag := field.Tag.Get(envConfigTagName)
		if _, ok := serversConfig[constants.ServerType(server)][envConfigTag]; !ok {
			return fmt.Errorf("%w: missing %s from %s.env, please add it", errInvalidEnv, envConfigTag, server)
		}

		delete(serversConfig[constants.ServerType(server)], envConfigTag)
	}

	if defaultStr := field.Tag.Get(defaultTagName); defaultStr != "" {
		return fmt.Errorf(
			"%w: has default value [%s]. You should add it to the corresponding env file and not here",
			errInvalidEnv,
			defaultStr,
		)
	}

	return nil
}
