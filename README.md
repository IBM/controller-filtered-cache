# controller-filtered-cache

A tool for customizing Kubernetes controller cache, using labels as filters to list and watch resources.

## Background

When an operator watches, lists or gets a Kubernetes resource type, the operator will store all the resources from this kind into its cache.

This will cause if there are a huge number of this kind of resource in the cluster, the operator will consume a large number of computing resources on the caching the Kubernetes resource that it won't use.

This controller-filtered-cache provides an implement for the operator to add a label selector to the operator cache. It will only store the resources with a specific label, which helps in reducing cache, memory footprint and CPU requirements.
.

## How to use controller-filtered-cache

[How to create filtered cache](docs/create-filtered-cache.md)

## Limitation

