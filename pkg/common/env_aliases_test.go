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
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/spf13/pflag"
)

var _ = Describe("SIM_ environment aliases", func() {
	baseArgs := []string{"cmd", "--model", TestModelName, "--mode", ModeRandom, "--seed", "100"}
	touched := []string{"SIM_MAX_MODEL_LEN", "SIM_PORT", "SIM_MAX_REQUEST_BODY_SIZE_MB", "SIM_CONFIG", "SIM_ENABLE_SLEEP_MODE"}

	clearEnv := func() {
		for _, name := range touched {
			Expect(os.Unsetenv(name)).To(Succeed())
		}
	}

	BeforeEach(clearEnv)
	AfterEach(clearEnv)

	It("Should name the variable after the flag", func() {
		Expect(EnvAliasName("max-model-len")).To(Equal("SIM_MAX_MODEL_LEN"))
		Expect(EnvAliasName("model")).To(Equal(ModelEnv))
	})

	It("Should apply an alias when the flag is not passed", func() {
		Expect(os.Setenv("SIM_MAX_MODEL_LEN", "4096")).To(Succeed())
		Expect(os.Setenv("SIM_MAX_REQUEST_BODY_SIZE_MB", "8")).To(Succeed())

		config, err := createSimConfig(baseArgs)
		Expect(err).NotTo(HaveOccurred())
		Expect(config.MaxModelLen).To(Equal(4096))
		Expect(config.MaxRequestBodySizeMB).To(Equal(8))
	})

	It("Should let the command line win over an alias", func() {
		Expect(os.Setenv("SIM_MAX_MODEL_LEN", "4096")).To(Succeed())

		config, err := createSimConfig(append(baseArgs, "--max-model-len", "2048"))
		Expect(err).NotTo(HaveOccurred())
		Expect(config.MaxModelLen).To(Equal(2048))
	})

	It("Should let an alias win over the yaml file", func() {
		Expect(os.Setenv("SIM_PORT", "9099")).To(Succeed())

		// manifests/config.yaml sets port 8001
		config, err := createSimConfig([]string{"cmd", "--config", "../../manifests/config.yaml"})
		Expect(err).NotTo(HaveOccurred())
		Expect(config.Port).To(Equal(9099))
	})

	It("Should ignore an empty alias rather than blank the value", func() {
		Expect(os.Setenv("SIM_MAX_MODEL_LEN", "")).To(Succeed())

		config, err := createSimConfig(baseArgs)
		Expect(err).NotTo(HaveOccurred())
		Expect(config.MaxModelLen).To(Equal(1024))
	})

	It("Should reject a value the flag cannot parse", func() {
		Expect(os.Setenv("SIM_PORT", "not-a-number")).To(Succeed())

		_, err := createSimConfig(baseArgs)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("SIM_PORT"))
	})

	It("Should not give the config-file flag an alias", func() {
		Expect(os.Setenv("SIM_CONFIG", "../../manifests/config.yaml")).To(Succeed())

		config, err := createSimConfig(baseArgs)
		Expect(err).NotTo(HaveOccurred())
		// the yaml would have set 8001
		Expect(config.Port).To(Equal(8000))
	})

	It("Should turn a toggle on and off", func() {
		enabled := true
		f := pflag.NewFlagSet("test", pflag.ContinueOnError)
		addToggle(f, &enabled, "enable-sleep-mode", "on", "off")

		Expect(os.Setenv("SIM_ENABLE_SLEEP_MODE", "false")).To(Succeed())
		Expect(applyEnvAliases(f)).To(Succeed())
		Expect(enabled).To(BeFalse(), "a false alias must reach the no- twin, not flip the toggle on")

		Expect(os.Setenv("SIM_ENABLE_SLEEP_MODE", "true")).To(Succeed())
		Expect(applyEnvAliases(f)).To(Succeed())
		Expect(enabled).To(BeTrue())
	})
})
