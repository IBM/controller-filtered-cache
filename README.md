# controller-filtered-cache

A tool for customizing Kubernetes controller cache, using labels as filters to list and watch resources.

## Background

When an operator watches, lists or gets a Kubernetes resource type, the operator will store all the resources from this kind into its cache.

This will cause if there are a huge number of this kind of resource in the cluster, the operator will consume a large number of computing resources on the caching the Kubernetes resource that it won't use.

This controller-filtered-cache provides an implement for the operator to add a label selector to the operator cache. It will only store the resources with a specific label, which helps in reducing cache, memory footprint and CPU requirements.

It supports the native kubernetes resources from GroupVersions: `corev1`, `appsv1`, `batchv1`, `certificatesv1beta1`, `corev1`, `networkingv1`, `rbacv1` and `storagev1`.

## How to use controller-filtered-cache

[How to create filtered cache](docs/create-filtered-cache.md)

## Limitation

1. The kubernetes selector doesn't support **OR** logic: For each kubernetes object, it only supports using one string as a selector for `LabelSelector` and `FieldSelector` respectively. It can't cache a resource from two different selectors. For example, users can't cache secrets with labels `app1` or `app2`.
