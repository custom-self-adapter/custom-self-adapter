/*
Copyright 2025 The Custom Self-Adapter Developers.

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

package evaluate

import (
	"github.com/custom-self-adapter/custom-self-adapter/metric"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// TODO Structurate the answer that the evaluation script will give.
// This will instruct CSA to select an adapt strategy by informing which
// adapt script to run.
type Evaluation struct {
}

type Info struct {
	Metrics    []*metric.ResourceMetric `json:"metrics"`
	Resource   metav1.Object            `json:"resource"`
	Evaluation *Evaluation              `json:"evaluation,omitempty"`
	RunType    string                   `json:"runType"`
}
