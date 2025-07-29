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

// Package selfadapter provides methods for adapting a resource by triggering
// metric gathering, feeding these metrics to an evaluation and feeding the
// evaluation result to a selected adapt strategy
package selfadapter

import (
	"fmt"

	"github.com/custom-self-adapter/custom-self-adapter/adapt"
	"github.com/custom-self-adapter/custom-self-adapter/config"
	"github.com/custom-self-adapter/custom-self-adapter/evaluate"
	"github.com/custom-self-adapter/custom-self-adapter/internal/adapting"
	"github.com/custom-self-adapter/custom-self-adapter/internal/evaluatecalc"
	"github.com/custom-self-adapter/custom-self-adapter/internal/metricget"
	"github.com/custom-self-adapter/custom-self-adapter/internal/resourceclient"
	"github.com/custom-self-adapter/custom-self-adapter/metric"
	"github.com/golang/glog"
	"k8s.io/apimachinery/pkg/labels"
)

type Adapter struct {
	Adapter         adapting.Adapter
	Client          resourceclient.Client
	Config          *config.Config
	GetMetricer     metricget.GetMetricer
	GetEvaluationer evaluatecalc.GetEvaluationer
}

func (a *Adapter) Adapt() error {
	glog.V(2).Infoln("Attempting to get managed resource")
	resource, err := a.Client.Get(a.Config.ScaleTargetRef.APIVersion, a.Config.ScaleTargetRef.Kind, a.Config.ScaleTargetRef.Name, a.Config.Namespace)
	if err != nil {
		return fmt.Errorf("failed to get managed resource: %w", err)
	}
	glog.V(2).Infof("Managed resource retrieved: %+v", resource)

	selector, err := labels.Parse(labels.FormatLabels(resource.GetLabels()))
	if err != nil {
		return fmt.Errorf("failed to parse selector for resource: %w", err)
	}
	glog.V(3).Infof("parsed selector for resource: %+v", selector)

	glog.V(2).Infoln("Attempting to get resource metrics")
	metrics, err := a.GetMetricer.GetMetrics(metric.Info{
		Resource: resource,
	}, selector)
	if err != nil {
		return fmt.Errorf("failed to get metrics: %w", err)
	}
	glog.V(2).Infof("Retrieved metrics: %+v", metrics)

	glog.V(2).Infoln("Attempting to evaluate metrics")
	evaluation, err := a.GetEvaluationer.GetEvaluation(evaluate.Info{
		Metrics:  metrics,
		Resource: resource,
	})
	if err != nil {
		return fmt.Errorf("failed to get evaluation: %w", err)
	}
	glog.V(2).Infof("Evaluation: %+v", evaluation)

	glog.V(2).Infoln("Attemping to execute adapt strategy")
	adaptation, err := a.Adapter.Adapt(adapt.Info{
		Evaluation: *evaluation,
		Resource:   resource,
	})
	if err != nil {
		return fmt.Errorf("failed to adapt resource: %w", err)
	}
	glog.V(2).Infof("Adapted resource successfully: %+v", adaptation)
	return nil
}
