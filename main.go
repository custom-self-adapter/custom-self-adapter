/*
Copyright 2021 The Custom Pod Autoscaler Authors.
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

// Custom Self-Adapter is the core program that runs inside a Custom Self-Adapter Image.
// The program handles interactions with the Kubernetes API, manages triggering Custom
// Self-Adapter User Logic through shell commands, exposes a simple HTTP REST API for viewing
// metrics and evaluations, and handles parsing user configuration to specify polling intervals,
// Kubernetes namespaces, command timeouts etc.
// The Custom Self-Adapter must be run inside a Kubernetes cluster.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	gohttp "net/http"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/custom-self-adapter/custom-self-adapter/internal/adapting"
	v1 "github.com/custom-self-adapter/custom-self-adapter/internal/api/v1"
	"github.com/custom-self-adapter/custom-self-adapter/internal/confload"
	"github.com/custom-self-adapter/custom-self-adapter/internal/evaluatecalc"
	"github.com/custom-self-adapter/custom-self-adapter/internal/execute"
	"github.com/custom-self-adapter/custom-self-adapter/internal/execute/http"
	"github.com/custom-self-adapter/custom-self-adapter/internal/execute/shell"
	"github.com/custom-self-adapter/custom-self-adapter/internal/metricget"
	"github.com/custom-self-adapter/custom-self-adapter/internal/resourceclient"
	"github.com/custom-self-adapter/custom-self-adapter/internal/selfadapter"
	"github.com/go-chi/chi"
	"github.com/golang/glog"
	"github.com/jthomperoo/k8shorizmetrics/v3"
	"github.com/jthomperoo/k8shorizmetrics/v3/metricsclient"
	"github.com/jthomperoo/k8shorizmetrics/v3/podsclient"
	"k8s.io/client-go/discovery"
	memory "k8s.io/client-go/discovery/cached"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
)

// Version is the version of the Custom Self-Adapter, injected in at build time
var Version = "development"

const configEnvName = "configPath"

const defaultConfig = "/config.yaml"

func init() {
	err := flag.Set("logtostderr", "true")
	if err != nil {
		log.Fatalf("Fail to set log to standard error flag: %s", err)
	}
	err = flag.Set("v", "0")
	if err != nil {
		log.Fatalf("Fail to set default log verbosity flag: %s", err)
	}
	flag.Parse()
}

func main() {
	// Read in environment variables
	configPath, exists := os.LookupEnv(configEnvName)
	if !exists {
		configPath = defaultConfig
	}

	// Convert all environment variables to a map
	configEnvs := map[string]string{}
	for _, envVar := range os.Environ() {
		i := strings.Index(envVar, "=")
		if i >= 0 {
			configEnvs[envVar[:i]] = envVar[i+1:]
		}
	}

	// Read in config file
	configFileData, err := os.ReadFile(configPath)
	if err != nil {
		glog.Fatalf("Fail to read configuration file: %s", err)
	}

	// Load Custom Pod Autoscaler config
	loadedConfig, err := confload.Load(configFileData, configEnvs)
	if err != nil {
		glog.Fatalf("Fail to parse configuration: %s", err)
	}

	// Create the in-cluster Kubernetes config
	clusterConfig, err := rest.InClusterConfig()
	if err != nil {
		glog.Fatalf("Fail to create in-cluster Kubernetes config: %s", err)
	}

	// Set up clientset
	clientset, err := kubernetes.NewForConfig(clusterConfig)
	if err != nil {
		glog.Fatalf("Fail to set up Kubernetes clientset: %s", err)
	}

	// Set up dynamic client
	dynamicClient, err := dynamic.NewForConfig(clusterConfig)
	if err != nil {
		glog.Fatalf("Fail to set up Kubernetes dynamic client: %s", err)
	}

	discoveryClient, err := discovery.NewDiscoveryClientForConfig(clusterConfig)
	if err != nil {
		glog.Fatalf("Fail to set up Kubernetes discovery client: %s", err)
	}
	cachedDiscoveryClient := memory.NewMemCacheClient(discoveryClient)

	// Set logging level
	err = flag.Lookup("v").Value.Set(strconv.Itoa(int(loadedConfig.LogVerbosity)))
	if err != nil {
		glog.Fatalf("Fail to set log verbosity: %s", err)
	}

	glog.V(0).Infof("Custom Self Adapter version: %s", Version)
	glog.V(1).Infoln("Setting up resources and clients")

	// Set up resource client
	resourceClient := &resourceclient.UnstructuredClient{
		Dynamic:    dynamicClient,
		RESTMapper: restmapper.NewDeferredDiscoveryRESTMapper(cachedDiscoveryClient),
	}

	// Create K8s metric gatherer, with required clients and configuration
	metricsclient := metricsclient.NewClient(clusterConfig, cachedDiscoveryClient)
	podsclient := &podsclient.OnDemandPodLister{
		Clientset: clientset,
	}
	cpuInitializationPeriod := time.Duration(loadedConfig.CPUInitializationPeriod) * time.Second
	initialReadinessDelay := time.Duration(loadedConfig.InitialReadinessDelay) * time.Second
	gatherer := k8shorizmetrics.NewGatherer(metricsclient, podsclient, cpuInitializationPeriod, initialReadinessDelay)

	// Set up shell executer
	shellExec := &shell.Execute{
		Command: exec.Command,
	}

	httpExec := http.DefaultExecute()

	// Combine executers
	combinedExecute := &execute.CombinedExecute{
		Executers: []execute.Executer{
			shellExec,
			httpExec,
		},
	}

	// Set up adapter
	adapter := &adapting.Adapt{
		Client:  *dynamicClient,
		Config:  loadedConfig,
		Execute: combinedExecute,
	}

	// Set up metric gathering
	metricGatherer := &metricget.Gatherer{
		Clientset:         clientset,
		Config:            loadedConfig,
		Execute:           combinedExecute,
		K8sMetricGatherer: gatherer,
	}

	// Set up evaluator
	evaluator := &evaluatecalc.Evaluator{
		Config:  loadedConfig,
		Execute: combinedExecute,
	}

	glog.V(1).Infoln("Setting up REST API")

	// Set up API
	api := &v1.API{
		Router:          chi.NewRouter(),
		Config:          loadedConfig,
		Client:          resourceClient,
		GetMetricer:     metricGatherer,
		GetEvaluationer: evaluator,
		Adapter:         adapter,
	}
	api.Routes()
	srv := gohttp.Server{
		Addr:    fmt.Sprintf("%s:%d", loadedConfig.APIConfig.Host, loadedConfig.APIConfig.Port),
		Handler: api.Router,
	}

	glog.V(1).Infoln("Setting up autoscaler")

	delayTime := loadedConfig.StartTime - (time.Now().UTC().UnixNano() / int64(time.Millisecond) % loadedConfig.StartTime)
	delayStartTimer := time.NewTimer(time.Duration(delayTime) * time.Millisecond)

	glog.V(0).Infof("Waiting %d milliseconds before starting autoscaler\n", delayTime)

	// Set up shutdown channel, which will listen for UNIX shutdown commands
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		// Wait for delay to start at expected time
		<-delayStartTimer.C
		glog.V(0).Infoln("Starting autoscaler")
		// Set up time ticker with configured interval
		ticker := time.NewTicker(time.Duration(loadedConfig.Interval) * time.Millisecond)

		// Set up adapter
		selfadapter := &selfadapter.Adapter{
			Client:          resourceClient,
			Config:          loadedConfig,
			GetMetricer:     metricGatherer,
			GetEvaluationer: evaluator,
			Adapter:         adapter,
		}

		// Run the scaler in a goroutine, triggered by the ticker
		// listen for shutdown requests, once received shut down the autoscaler
		// and the API
		go func() {
			for {
				select {
				case <-shutdown:
					glog.V(0).Infoln("Shutting down...")
					// Stop API
					ctx, cancel := context.WithCancel(context.Background())
					defer cancel()
					srv.Shutdown(ctx)
					// Stop autoscaler
					ticker.Stop()
					return
				case <-ticker.C:
					glog.V(2).Infoln("Running autoscaler")
					err := selfadapter.Adapt()
					if err != nil {
						glog.Errorf("Error while autoscaling: %s", err)
					}
				}
			}
		}()
	}()

	if loadedConfig.APIConfig.Enabled {
		if loadedConfig.APIConfig.UseHTTPS {
			glog.V(0).Infoln("Starting API using HTTPS")
			// Start API
			err := srv.ListenAndServeTLS(loadedConfig.APIConfig.CertFile, loadedConfig.APIConfig.KeyFile)
			if err != gohttp.ErrServerClosed {
				glog.Fatalf("HTTPS API Error: %s", err)
			}

		} else {
			glog.V(0).Infoln("Starting API using HTTP")
			// Start API
			err := srv.ListenAndServe()
			if err != gohttp.ErrServerClosed {
				glog.Fatalf("HTTP API Error: %s", err)
			}
		}
	} else {
		glog.V(0).Infoln("API disabled, skipping starting the API")
		<-shutdown
	}
}
