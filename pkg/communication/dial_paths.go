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

package communication

import "github.com/buaazp/fasthttprouter"

// registerDialRoutes aliases the ingress paths DIAL Core forwards on to the handlers that
// already serve their standard counterparts. Nothing else changes: the handlers read the
// model from the request body, so the :model path segment is ignored, exactly as it is by
// the providers DIAL addresses this way.
//
// DIAL Core appends the inbound ingress path to a deployment's interfaces base_url
// (DeploymentEndpointUtil.resolveRequestUri), so a target reachable through the interfaces
// map has to serve the DIAL path rather than the API's own. Without these aliases the
// simulator is reachable only through the legacy complete-url fields (endpoint,
// responsesEndpoint), which cannot express anthropicMessages at all and cannot express
// chat completions and embeddings at the same time.
//
// Off by default: pass --dial-paths (or dial-paths: true) to enable.
func (c *Communication) registerDialRoutes(r *fasthttprouter.Router) {
	if !c.runtime.Config().DialPaths {
		return
	}

	// named in the request body, no deployment segment
	r.POST("/anthropic/v1/messages", c.HandleMessages)
	r.POST("/openai/v1/responses", c.HandleResponses)

	// the deployments-POST family; Core rewrites :model to the deployment's overrideName
	r.POST("/openai/deployments/:model/chat/completions", c.HandleChatCompletions)
	r.POST("/openai/deployments/:model/completions", c.HandleTextCompletions)
	if !c.runtime.Config().MMEncoderOnly {
		r.POST("/openai/deployments/:model/embeddings", c.HandleEmbeddings)
	}
}
