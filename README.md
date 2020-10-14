# controller-filtered-cache

A tool for customizing Kubernetes controller cache, using labels as filters to list and watch resources.

## Background

When an operator watches, lists or gets a Kubernetes resource type, the operator will store all the resources from this kind into its cache.

This will cause if there are a huge number of this kind of resource in the cluster, the operator will consume a large number of computing resources on the caching the Kubernetes resource that it won't use.

This controller-filtered-cache provides an implement for the operator to add a label selector to the operator cache. It will only store the resources with a specific label, which helps in reducing cache, memory footprint and CPU requirements.

## How to use controller-filtered-cache to customize operator cache

The controller-filtered-cache is used to customize operator cache when initialize the operator manager.

1. Import the library

    Add the `controller-filtered-cache` as a Golang library

    ```yaml
        cache "github.com/IBM/controller-filtered-cache/filteredcache"
    ```

1. Create a map for GVKs and labels

    This map is used to set the Kubernetes resource and the label applied to the cache.

    ```yaml
    gvkLabelMap := map[schema.GroupVersionKind]string{
        corev1.SchemeGroupVersion.WithKind("Secret"):    "managed-by-controller",
        corev1.SchemeGroupVersion.WithKind("ConfigMap"): "managed-by-controller",
    }
    ```

    The above example means the operator cache will only store the `Secret` and `ConfigMap` with label `managed-by-controller`.

    **Note:** `corev1` in the above example is from `corev1 "k8s.io/api/core/v1"`. The controller-filtered-cache only supports the resources from `core` api group.

1. Add the customized cache into the operator manager

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

    Using `NewFilteredCacheBuilder` function to create a customized map based on `gvkLabelMap` we created in Step 2.

## Limitation

1. It only supports `labelSelector` but not the `fieldSelector`.

1. Since the controller-filtered-cache uses the `clientSet.CoreV1().RESTClient()` as client to get and list resource, controller-filtered-cache can only support the resource from the core api group, like `Pod`, `ConfigMap` and `Secret`. For the other resources, like `deployment` and `job`, users can customize your own filtered cache based on `filteredcache/filtered-cache.go` file.
