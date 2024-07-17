// Copyright (c) 2024 Trenova Technologies, LLC
//
// Licensed under the Business Source License 1.1 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://trenova.app/pricing/
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//
// Key Terms:
// - Non-production use only
// - Change Date: 2026-11-16
// - Change License: GNU General Public License v2 or later
//
// For full license text, see the LICENSE file in the root directory.

package types

import "github.com/bytedance/sonic"

type InvalidParam struct {
	Name   string `json:"name"`
	Reason string `json:"reason"`
}

type ProblemDetail struct {
	Type          string         `json:"type"`
	Title         string         `json:"title"`
	Status        int            `json:"status"`
	Detail        string         `json:"detail"`
	Instance      string         `json:"instance,omitempty"`
	InvalidParams []InvalidParam `json:"invalid-params,omitempty"`
}

func (p *ProblemDetail) Error() string {
	errBytes, _ := sonic.Marshal(p)
	return string(errBytes)
}
