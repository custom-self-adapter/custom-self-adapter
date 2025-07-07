# Custom Self-Adapter

Custom Self-Adapters (CSAs) are custom Kubernetes self-adapters. They act on metrics and act on workloads to configure them using script written by you in any language.

This project is part of a framework that lets you build self-adapters for Kubernetes using the language of your choice. It's strongly based on the [Custom Pod Autoscaler](https://custom-pod-autoscaler.readthedocs.io/en/stable/) functionality and code.

# Why woul i use it?

Kubernetes provides the Horizontal Pod Autoscaler, which reacts to metrics by scaling the number or replicas in a resource (Deployment, ReplicationController, ReplicaSet, StatefulSet). Its limitation is that it has a hard-coded algorithm for calculating how many replicas are needed:

```
desiredReplicas = ceil[currentReplicas * ( currentMetricValue / desiredMetricValue )]
```

The Custom Pod Autoscaler works by letting the user write their own scripts to collecting metrics (metrics scripts) and to evaluate the desired number of replicas (evaluation scripts), but it still limited to changing the number of replicas in the managed resource.

The Custom Self-Adapter extends the Custom Pod Autoscaler logic by letting the user write their script to also act on the cluster, modifying any resource that would allow the managed application to better respond to their users. Those are the adapt scripts.