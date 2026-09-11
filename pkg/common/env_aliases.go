/*
Copyright 2026 The llm-d-inference-sim Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package common

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/pflag"
)

// EnvAliasPrefix is prepended to a flag name to form its environment-variable alias:
// --max-model-len is SIM_MAX_MODEL_LEN, --log-http is SIM_LOG_HTTP.
const EnvAliasPrefix = "SIM_"

// envAliasSkipped are flags with no environment alias.
//
// The first four are parsed straight from os.Args before the flag set exists
// (getParamValueFromArgs), so setting them here would be read by nothing and look like it
// worked. The rest are meta flags that only make sense on a command line.
var envAliasSkipped = map[string]bool{
	"config":            true,
	"served-model-name": true,
	"lora-modules":      true,
	"fake-metrics":      true,
	"help":              true,
}

// EnvAliasName is the environment variable read for a flag: "-" becomes "_", upper-cased,
// behind EnvAliasPrefix.
func EnvAliasName(flagName string) string {
	return EnvAliasPrefix + strings.ToUpper(strings.ReplaceAll(flagName, "-", "_"))
}

// applyEnvAliases fills in every flag that was not passed on the command line from its
// SIM_<FLAG_NAME> environment variable, so that a deployment can be configured by
// environment alone. A flag that was passed always wins; an empty variable is ignored, so
// exporting SIM_FOO="" cannot blank a YAML value by accident.
//
// Must run after f.Parse and after the YAML file has been merged: a flag's current value is
// the YAML one at that point, and an alias is meant to override it. That places the aliases
// between the command line and the YAML file in the precedence order documented in
// docs/configuration.md.
//
// An unparseable value is an error rather than a warning — a mistyped SIM_PORT should not
// leave the simulator listening somewhere unintended.
func applyEnvAliases(f *pflag.FlagSet) error {
	var err error

	f.VisitAll(func(fl *pflag.Flag) {
		if err != nil || fl.Changed || envAliasSkipped[fl.Name] || strings.HasPrefix(fl.Name, "no-") {
			return
		}

		name := EnvAliasName(fl.Name)
		value, found := os.LookupEnv(name)
		if !found || value == "" {
			return
		}

		if setErr := setFlagFromEnv(f, fl, value); setErr != nil {
			err = fmt.Errorf("%s=%q is not a valid value for --%s: %w", name, value, fl.Name, setErr)
		}
	})

	return err
}

// setFlagFromEnv assigns value to fl.
//
// Boolean flags need care. Some of them are registered by addToggle as a pair of flags
// sharing one variable, where Set ignores its argument and applies a hardcoded true or
// false — so Set("false") on the positive flag of such a pair would switch the setting *on*.
// For a false value we therefore prefer the "no-" twin when the pair exists, and only fall
// back to Set on the flag itself for an ordinary boolean.
func setFlagFromEnv(f *pflag.FlagSet, fl *pflag.Flag, value string) error {
	if fl.Value.Type() != "bool" {
		return fl.Value.Set(value)
	}

	enabled, err := strconv.ParseBool(value)
	if err != nil {
		return err
	}
	if !enabled {
		if twin := f.Lookup("no-" + fl.Name); twin != nil {
			return twin.Value.Set("true")
		}
	}
	return fl.Value.Set(strconv.FormatBool(enabled))
}
