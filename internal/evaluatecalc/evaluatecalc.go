/*
Copyright 2021 The Custom Pod Autoscaler Authors.

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

// Package evaluatecalc provides functionality for managing evaluating,
// calling external evaluation logic through shell commands with
// relevant data piped to them.
package evaluatecalc

import (
	"encoding/json"
	"fmt"

	"github.com/custom-self-adapter/custom-self-adapter/config"
	"github.com/custom-self-adapter/custom-self-adapter/evaluate"
	"github.com/custom-self-adapter/custom-self-adapter/internal/execute"
	"github.com/golang/glog"
)

// GetEvaluationer provides methods for retrieving an evaluation
type GetEvaluationer interface {
	GetEvaluation(info evaluate.Info) (*evaluate.Evaluation, error)
}

// Evaluator handles triggering the evaluation logic to decide how to scale a resource
type Evaluator struct {
	Config  *config.Config
	Execute execute.Executer
}

// GetEvaluation uses the metrics provided to determine a set of evaluations
func (e *Evaluator) GetEvaluation(info evaluate.Info) (*evaluate.Evaluation, error) {
	glog.V(3).Infoln("Evaluating gathered metrics")
	// Convert evaluation input into JSON
	specJSON, err := json.Marshal(info)
	if err != nil {
		// Should not occur, panic
		panic(err)
	}

	if e.Config.PreEvaluate != nil {
		glog.V(3).Infoln("Attempting to run pre-evaluate hook")
		hookResult, err := e.Execute.ExecuteWithValue(e.Config.PreEvaluate, string(specJSON))
		if err != nil {
			return nil, fmt.Errorf("failed run pre-evaluate hook: %w", err)
		}
		glog.V(3).Infof("Pre-evaluate hook response: %+v", hookResult)
	}

	glog.V(3).Infoln("Attempting to run evaluation logic")
	gathered, err := e.Execute.ExecuteWithValue(e.Config.Evaluate, string(specJSON))
	if err != nil {
		return nil, fmt.Errorf("failed to get evaluation: %w", err)
	}
	glog.V(3).Infof("Evaluation determined: %s", gathered)

	glog.V(3).Infoln("Attempting to parse evaluation")
	evaluation := &evaluate.Evaluation{}
	err = json.Unmarshal([]byte(gathered), evaluation)
	if err != nil {
		return nil, fmt.Errorf("failed to parse JSON evaluation, got '%s': %w", gathered, err)
	}
	glog.V(3).Infof("Evaluation parsed: %+v", evaluation)

	// Add results into the evaluation spec
	info.Evaluation = evaluation

	if e.Config.PostEvaluate != nil {
		glog.V(3).Infoln("Attempting to run post-evaluate hook")
		// Convert post evaluation into JSON
		postSpecJSON, err := json.Marshal(info)
		if err != nil {
			// Should not occur, panic
			panic(err)
		}
		hookResult, err := e.Execute.ExecuteWithValue(e.Config.PostEvaluate, string(postSpecJSON))
		if err != nil {
			return nil, fmt.Errorf("failed run post-evaluate hook: %w", err)
		}
		glog.V(3).Infof("Post-evaluate hook response: %+v", hookResult)
	}
	return evaluation, nil
}
