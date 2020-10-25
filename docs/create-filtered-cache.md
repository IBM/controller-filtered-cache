<!-- START doctoc generated TOC please keep comment here to allow auto update -->
<!-- DON'T EDIT THIS SECTION, INSTEAD RE-RUN doctoc TO UPDATE -->
**Table of Contents**  *generated with [DocToc](https://github.com/thlorenz/doctoc)*

- [How to use controller-filtered-cache to customize operator cache](#how-to-use-controller-filtered-cache-to-customize-operator-cache)
  - [1. Import the library](#1-import-the-library)
  - [2. Create a map for GVKs and selector](#2-create-a-map-for-gvks-and-selector)
  - [3. Create customized cache when initializing the operator manager](#3-create-customized-cache-when-initializing-the-operator-manager)
    - [Single namespace and all namespaces operator](#single-namespace-and-all-namespaces-operator)
    - [multiple namespaces operator](#multiple-namespaces-operator)

<!-- END doctoc generated TOC please keep comment here to allow auto update -->

# How to use controller-filtered-cache to customize operator cache

The controller-filtered-cache is used to customize operator cache when initializing the operator manager.

## 1. Import the library

Add the `controller-filtered-cache` as a Golang library into `go.mod`

If the operator is running `controller-runtime v0.5.0`, please use `v0.1.x`

```go.sum
    github.com/IBM/controller-filtered-cache v0.1.1
```

If the operator is running `controller-runtime v0.6.0`, please use `v0.2.x`

```go.sum
    github.com/IBM/controller-filtered-cache v0.2.0
```

import the `controller-filtered-cache` into `main.go`

```yaml
    "github.com/IBM/controller-filtered-cache/filteredcache"
```

## 2. Create a map for GVKs and selector

This map is used to set the Kubernetes resource and the label applied to the cache.

```yaml
gvkLabelMap := map[schema.GroupVersionKind]filteredcache.Selector{
    corev1.SchemeGroupVersion.WithKind("Secret"): filteredcache.Selector{
        FieldSelector: "metadata.name==mongodb-admin",
        LabelSelector: "managedBy",
    },
    corev1.SchemeGroupVersion.WithKind("ConfigMap"): filteredcache.Selector{
        LabelSelector: "managedBy==mongodb",
    },
    appsv1.SchemeGroupVersion.WithKind("Deployment"): filteredcache.Selector{
        LabelSelector: "app",
    },
}
```

`filteredcache.Selector` is a structure that contains two fields `FieldSelector` and `LabelSelector`. Uses can input the stringified selector to specify the resource should be cached. For more information, please refer Kubernetes documentation [Label-selectors](https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/) and [field-selectors](https://kubernetes.io/docs/concepts/overview/working-with-objects/field-selectors/)

The above example means the operator cache will only store the `Secret` that exists label `managedBy` and its name equals to `mongodb-admin`, `ConfigMap` that has the label `managedBy: mongodb` and `Deployment` exists label `app`.

**Note:** In the above example, `corev1` is from `corev1 "k8s.io/api/core/v1"`, appsv1 is from `appsv1 "k8s.io/api/apps/v1"` and `schema` is from `"k8s.io/apimachinery/pkg/runtime/schema"`

## 3. Create customized cache when initializing the operator manager

### Single namespace and all namespaces operator

Using `NewFilteredCacheBuilder` function to create a customized cache based on `gvkLabelMap` we created in Step 2.

**kubebuilder v2 or operator-sdk v0.19.0+:**

```yaml
mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
    Scheme:             scheme,
    MetricsBindAddress: metricsAddr,
    Port:               9443,
    LeaderElection:     enableLeaderElection,
    LeaderElectionID:   "2e672f4a.ibm.com",
    NewCache:           cache.NewFilteredCacheBuilder(gvkLabelMap),
})
```

**operator-sdk v0.15.0-v0.18.2:**

```yaml
mgr, err := manager.New(cfg, manager.Options{
    Namespace:          namespace,
    MetricsBindAddress: fmt.Sprintf("%s:%d", metricsHost, metricsPort),
    NewCache:           cache.NewFilteredCacheBuilder(gvkLabelMap),
})
```

### multiple namespaces operator

Using `MultiNamespacedFilteredCacheBuilder` function to create a customized cache and manage resources in a set of Namespaces.

**kubebuilder v2 or operator-sdk v0.19.0+:**

```yaml
...
namespaces := []string{"foo", "bar"} // List of Namespaces
...
mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
    Scheme:             scheme,
    MetricsBindAddress: metricsAddr,
    Port:               9443,
    LeaderElection:     enableLeaderElection,
    LeaderElectionID:   "2e672f4a.ibm.com",
    NewCache:           cache.NewFilteredCacheBuilder(gvkLabelMap, namespaces),
})
```

**operator-sdk v0.15.0-v0.18.2:**

```yaml
...
namespaces := []string{"foo", "bar"} // List of Namespaces
...
mgr, err := manager.New(cfg, manager.Options{
    Namespace:          namespace,
    MetricsBindAddress: fmt.Sprintf("%s:%d", metricsHost, metricsPort),
    NewCache:           cache.NewFilteredCacheBuilder(gvkLabelMap, namespaces),
})
```
