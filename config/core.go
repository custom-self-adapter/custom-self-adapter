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

package config

import (
	autoscaling "k8s.io/api/autoscaling/v2"
)

const (
	// APIRunType marks the metric gathering/evaluation as running during an API
	// request, which will use the results to adapt
	APIRunType = "api"
	// APIDryRunRunType marks the metric gathering/evaluation as running during an
	// API request, which will only view the results and not use it for adapt
	APIDryRunRunType = "api_dry_run"
	// AdaptRunType marks the metric gathering/evaluation as running during an
	// adapt
	AdaptRunType = "adapt"
)

const (
	// DefaultInterval is the default interval value
	DefaultInterval = 15000
	// DefaultNamespace is the default namespace value
	DefaultNamespace = "default"
	// DefaultStartTime is the default start time
	DefaultStartTime = 1
	// DefaultLogVerbosity is the default log verbosity
	DefaultLogVerbosity = 0
	// DefaultDownscaleStabilization is the default downscale stabilization value
	DefaultDownscaleStabilization = 0
	// DefaultCPUInitializationPeriod is the default CPU initialization value
	DefaultCPUInitializationPeriod = 300
	// DefaultInitialReadinessDelay is the default initial readiness delay value
	DefaultInitialReadinessDelay = 30
)

const (
	// DefaultAPIEnabled is the default value for the API being enabled
	DefaultAPIEnabled = true
	// DefaultUseHTTPS is the default value for the API using HTTPS
	DefaultUseHTTPS = false
	// DefaultHost is the default address for the API
	DefaultHost = "0.0.0.0"
	// DefaultPort is the default port for the API
	DefaultPort = 5000
	// DefaultCertFile is the default cert file for the API
	DefaultCertFile = ""
	// DefaultKeyFile is the default private key file for the API
	DefaultKeyFile = ""
)

type Config struct {
	ScaleTargetRef           *autoscaling.CrossVersionObjectReference `json:"scaleTargetRef"`
	PreMetric                *Method                                  `json:"preMetric"`
	PostMetric               *Method                                  `json:"postMetric"`
	PreEvaluate              *Method                                  `json:"preEvaluate"`
	PostEvaluate             *Method                                  `json:"postEvaluate"`
	PreAdapt                 *Method                                  `json:"preAdapt"`
	PostAdapt                *Method                                  `json:"postAdapt"`
	Evaluate                 *Method                                  `json:"evaluate"`
	Metric                   *Method                                  `json:"metric"`
	Adapt                    map[string]*Method                       `json:"adapt"`
	Interval                 int                                      `json:"interval"`
	Namespace                string                                   `json:"namespace"`
	StartTime                int64                                    `json:"startTime"`
	LogVerbosity             int32                                    `json:"logVerbosity"`
	DownscaleStabilization   int                                      `json:"downscaleStabilization"`
	APIConfig                *APIConfig                               `json:"apiConfig"`
	KubernetesMetricSpecs    []K8sMetricSpec                          `json:"kubernetesMetricSpecs"`
	RequireKubernetesMetrics bool                                     `json:"requireKubernetesMetrics"`
	InitialReadinessDelay    int64                                    `json:"initialReadinessDelay"`
	CPUInitializationPeriod  int64                                    `json:"cpuInitializationPeriod"`
}

func NewConfig() *Config {
	return &Config{
		Interval:               DefaultInterval,
		Namespace:              DefaultNamespace,
		StartTime:              DefaultStartTime,
		DownscaleStabilization: DefaultDownscaleStabilization,
		APIConfig: &APIConfig{
			Enabled:  DefaultAPIEnabled,
			UseHTTPS: DefaultUseHTTPS,
			Port:     DefaultPort,
			Host:     DefaultHost,
			CertFile: DefaultCertFile,
			KeyFile:  DefaultKeyFile,
		},
		KubernetesMetricSpecs:    nil,
		RequireKubernetesMetrics: false,
		InitialReadinessDelay:    DefaultInitialReadinessDelay,
		CPUInitializationPeriod:  DefaultCPUInitializationPeriod,
	}
}
