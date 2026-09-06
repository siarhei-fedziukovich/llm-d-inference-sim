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

package tests

import (
	"context"
	"net/http"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/llm-d/llm-d-inference-sim/pkg/common"
)

// The DIAL ingress paths and a body each one accepts.
var dialPathCases = []struct {
	path string
	body string
}{
	{
		"/anthropic/v1/messages",
		`{"model":"` + common.TestModelName + `","messages":[{"role":"user","content":"Hello"}],"max_tokens":10}`,
	},
	{
		"/openai/v1/responses",
		`{"model":"` + common.TestModelName + `","input":[{"type":"message","role":"user",` +
			`"content":[{"type":"input_text","text":"Hello"}]}],"max_output_tokens":10}`,
	},
	{
		"/openai/deployments/" + common.TestModelName + "/chat/completions",
		`{"model":"` + common.TestModelName + `","messages":[{"role":"user","content":"Hello"}],"max_tokens":10}`,
	},
	{
		"/openai/deployments/" + common.TestModelName + "/completions",
		`{"model":"` + common.TestModelName + `","prompt":"Hello","max_tokens":10}`,
	},
	{
		"/openai/deployments/" + common.TestModelName + "/embeddings",
		`{"model":"` + common.TestModelName + `","input":["Hello"]}`,
	},
}

func postDialPath(client *http.Client, path string, body string) int {
	resp, err := client.Post("http://localhost"+path, "application/json", strings.NewReader(body))
	Expect(err).NotTo(HaveOccurred())
	defer resp.Body.Close() //nolint:errcheck
	return resp.StatusCode
}

var _ = Describe("DIAL ingress paths", func() {
	It("Should serve the DIAL paths when --dial-paths is set", func() {
		client, err := startServerWithArgs(context.TODO(),
			[]string{"cmd", "--model", common.TestModelName, "--mode", common.ModeRandom, "--dial-paths"})
		Expect(err).NotTo(HaveOccurred())

		for _, c := range dialPathCases {
			Expect(postDialPath(client, c.path, c.body)).To(Equal(http.StatusOK), "path: %s", c.path)
		}
	})

	It("Should keep serving the standard paths when --dial-paths is set", func() {
		client, err := startServerWithArgs(context.TODO(),
			[]string{"cmd", "--model", common.TestModelName, "--mode", common.ModeRandom, "--dial-paths"})
		Expect(err).NotTo(HaveOccurred())

		body := `{"model":"` + common.TestModelName + `","messages":[{"role":"user","content":"Hello"}],"max_tokens":10}`
		Expect(postDialPath(client, "/v1/chat/completions", body)).To(Equal(http.StatusOK))
		Expect(postDialPath(client, "/v1/messages", body)).To(Equal(http.StatusOK))
	})

	It("Should not serve the DIAL paths by default", func() {
		client, err := startServerWithArgs(context.TODO(),
			[]string{"cmd", "--model", common.TestModelName, "--mode", common.ModeRandom})
		Expect(err).NotTo(HaveOccurred())

		for _, c := range dialPathCases {
			Expect(postDialPath(client, c.path, c.body)).To(Equal(http.StatusNotFound), "path: %s", c.path)
		}
	})
})
