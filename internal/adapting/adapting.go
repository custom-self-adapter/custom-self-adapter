/*
Copyright 2025 The Custom Self-Adapter Authors.

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

// Package adapt provides functionality for adapting Kubernetes resources,
// calling external adapt scripts though shell commands with relevant
// data piped to them.
package adapting

import (
	"encoding/json"
	"fmt"

	"github.com/custom-self-adapter/custom-self-adapter/adapt"
	"github.com/custom-self-adapter/custom-self-adapter/config"
	"github.com/custom-self-adapter/custom-self-adapter/internal/execute"
	"github.com/golang/glog"
	"k8s.io/client-go/dynamic"
)

type Adapter interface {
	Adapt(info adapt.Info) (*adapt.Adaptation, error)
}

type Adapt struct {
	Client  dynamic.DynamicClient
	Config  *config.Config
	Execute execute.Executer
}

func (a *Adapt) Adapt(info adapt.Info) (*adapt.Adaptation, error) {
	glog.V(3).Infof("Executing the adaptation strategy: %w", info.Evaluation.Strategy)
	specJSON, err := json.Marshal(info)
	if err != nil {
		panic(err)
	}

	strategy, exists := a.Config.Adapt[info.Evaluation.Strategy]
	if !exists {
		return nil, fmt.Errorf("Strategy does not exists: %v", strategy)
	}

	glog.V(3).Infoln("Attempting to run adaptation logic")
	result, err := a.Execute.ExecuteWithValue(strategy, string(specJSON))
	if err != nil {
		return nil, fmt.Errorf("failed to run adaptation %w", err)
	}
	glog.V(3).Infof("Adaptation result %s", result)

	glog.V(3).Infoln("Attempting to parse adaptation")
	adaptation := &adapt.Adaptation{}
	err = json.Unmarshal([]byte(result), adaptation)
	if err != nil {
		return nil, fmt.Errorf("falied to parse JSON adaptation, got '%s', %w", result, err)
	}

	info.Adaptation = adaptation

	return adaptation, nil
}
