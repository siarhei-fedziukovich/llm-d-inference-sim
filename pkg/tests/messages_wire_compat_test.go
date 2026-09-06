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
	"encoding/json"
	"net/http"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/llm-d/llm-d-inference-sim/pkg/api"
	"github.com/llm-d/llm-d-inference-sim/pkg/common"
)

// Wire-format details of /v1/messages that real Anthropic clients depend on.
// Both were found by putting a real DIAL adapter in front of the simulator.
var _ = Describe("Messages API wire compatibility", func() {

	It("accepts a system prompt given as an array of content blocks", func() {
		client, err := startServer(context.TODO(), common.ModeRandom)
		Expect(err).NotTo(HaveOccurred())

		body := `{"model":"` + common.TestModelName + `",` +
			`"system":[{"type":"text","text":"You are terse."}],` +
			`"messages":[{"role":"user","content":"Hello"}],"max_tokens":10}`

		resp, err := client.Post("http://localhost/v1/messages", "application/json", strings.NewReader(body))
		Expect(err).NotTo(HaveOccurred())
		defer resp.Body.Close() //nolint:errcheck
		Expect(resp.StatusCode).To(Equal(http.StatusOK))
	})

	It("still accepts a system prompt given as a plain string", func() {
		client, err := startServer(context.TODO(), common.ModeRandom)
		Expect(err).NotTo(HaveOccurred())

		body := buildMessagesBody(common.TestModelName, false,
			[]map[string]any{{"role": "user", "content": "Hello"}},
			"You are terse.", nil, nil)
		Expect(sendMessagesRequest(client, body)).NotTo(BeNil())
	})

	It("sends an empty text field on content_block_start", func() {
		client, err := startServer(context.TODO(), common.ModeRandom)
		Expect(err).NotTo(HaveOccurred())

		body := buildMessagesBody(common.TestModelName, true,
			[]map[string]any{{"role": "user", "content": "Hello"}}, "", nil, nil)

		var starts int
		for _, event := range readMessagesSSEStream(client, body) {
			if event.EventType != api.MessagesEventContentBlockStart {
				continue
			}
			starts++

			// The raw JSON must carry the field: a client that accumulates text
			// deltas onto this block starts from null when it is omitted.
			Expect(string(event.Data)).To(ContainSubstring(`"text":""`))

			var parsed struct {
				ContentBlock struct {
					Type string  `json:"type"`
					Text *string `json:"text"`
				} `json:"content_block"`
			}
			Expect(json.Unmarshal(event.Data, &parsed)).To(Succeed())
			Expect(parsed.ContentBlock.Type).To(Equal("text"))
			Expect(parsed.ContentBlock.Text).NotTo(BeNil())
			Expect(*parsed.ContentBlock.Text).To(Equal(""))
		}
		Expect(starts).To(Equal(1), "expected exactly one content_block_start for a text response")
	})
})
