//
// Copyright 2020 IBM Corporation
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//

package cache

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/gobuffalo/flect"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog"
	cr_cache "sigs.k8s.io/controller-runtime/pkg/cache"
	cr_client "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
)

// ErrUnsupported is returned for unsupported operations
var ErrUnsupported = errors.New("unsupported operation")

// ErrInternalError is returned for unexpected errors
var ErrInternalError = errors.New("internal error")

// NewFilteredCacheBuilder create a customized cache
func NewFilteredCacheBuilder(gvkLabelMap map[schema.GroupVersionKind]string) cr_cache.NewCacheFunc {
	return func(config *rest.Config, opts cr_cache.Options) (cr_cache.Cache, error) {
		// Setup filtered informer that will only store/return items matching the filter for listing purposes
		clientSet, err := kubernetes.NewForConfig(config)
		if err != nil {
			return nil, err
		}

		var resync time.Duration
		if opts.Resync != nil {
			resync = *opts.Resync
		}

		informerMap := buildInformerMap(clientSet, opts, gvkLabelMap, resync)

		fallback, err := cr_cache.New(config, opts)
		if err != nil {
			klog.Error(err, "Failed to init fallback cache")
			return nil, err
		}
		return filteredCache{clientSet: clientSet, informerMap: informerMap, fallback: fallback, Scheme: opts.Scheme}, nil
	}
}

func buildInformerMap(clientSet *kubernetes.Clientset, opts cr_cache.Options, gvkLabelMap map[schema.GroupVersionKind]string, resync time.Duration) map[schema.GroupVersionKind]cache.SharedIndexInformer {
	informerMap := make(map[schema.GroupVersionKind]cache.SharedIndexInformer)
	for gvk, label := range gvkLabelMap {
		plural := strings.ToLower(flect.Pluralize(gvk.Kind))
		listerWatcher := cache.NewFilteredListWatchFromClient(clientSet.CoreV1().RESTClient(), plural, opts.Namespace, func(options *metav1.ListOptions) {
			options.LabelSelector = label
		})

		objType := &unstructured.Unstructured{}
		objType.GetObjectKind().SetGroupVersionKind(gvk)

		informer := cache.NewSharedIndexInformer(listerWatcher, objType, resync, cache.Indexers{})

		informerMap[gvk] = informer
		gvkList := schema.GroupVersionKind{Group: gvk.Group, Version: gvk.Version, Kind: gvk.Kind + "List"}
		informerMap[gvkList] = informer
	}
	return informerMap
}

type filteredCache struct {
	clientSet   *kubernetes.Clientset
	informerMap map[schema.GroupVersionKind]cache.SharedIndexInformer
	fallback    cr_cache.Cache
	Scheme      *runtime.Scheme
}

func (cc filteredCache) Get(ctx context.Context, key cr_client.ObjectKey, obj runtime.Object) error {
	gvk, err := apiutil.GVKForObject(obj, cc.Scheme)
	if err != nil {
		klog.Error(err)
		return err
	}
	if informer, ok := cc.informerMap[gvk]; ok {
		if err := cc.getFromStore(informer, key, obj); err == nil {
		} else if err := cc.getFromClient(ctx, key, obj); err != nil {
			klog.Error(err)
			return err
		}
		return nil
	}

	// Passthrough
	return cc.fallback.Get(ctx, key, obj)
}

func (cc filteredCache) getFromStore(informer cache.SharedIndexInformer, key cr_client.ObjectKey, obj runtime.Object) error {
	gvk, err := apiutil.GVKForObject(obj, cc.Scheme)
	if err != nil {
		return err
	}

	var keyString string
	if key.Namespace == "" {
		keyString = key.Name
	} else {
		keyString = key.Namespace + "/" + key.Name
	}
	item, exists, err := informer.GetStore().GetByKey(keyString)
	if err != nil {
		klog.Info("Failed to get item from cache", "error", err)
		return ErrInternalError
	}
	if !exists {
		return apierrors.NewNotFound(schema.GroupResource{Group: gvk.Group, Resource: gvk.Kind}, key.String())
	}
	if _, isObj := item.(runtime.Object); !isObj {
		// This should never happen
		return fmt.Errorf("cache contained %T, which is not an Object", item)
	}

	// deep copy to avoid mutating cache
	item = item.(runtime.Object).DeepCopyObject()

	// Copy the value of the item in the cache to the returned value
	objVal := reflect.ValueOf(obj)
	itemVal := reflect.ValueOf(item)

	if !objVal.Type().AssignableTo(objVal.Type()) {
		return fmt.Errorf("cache had type %s, but %s was asked for", itemVal.Type(), objVal.Type())
	}
	reflect.Indirect(objVal).Set(reflect.Indirect(itemVal))
	obj.GetObjectKind().SetGroupVersionKind(gvk)

	return nil
}

func (cc filteredCache) getFromClient(ctx context.Context, key cr_client.ObjectKey, obj runtime.Object) error {

	gvk, err := apiutil.GVKForObject(obj, cc.Scheme)
	if err != nil {
		return err
	}

	resource := kindToResource(gvk.Kind)
	result, err := cc.clientSet.CoreV1().RESTClient().
		Get().
		Namespace(key.Namespace).
		Name(key.Name).
		Resource(resource).
		VersionedParams(&metav1.GetOptions{}, metav1.ParameterCodec).
		Do(ctx).
		Get()

	if apierrors.IsNotFound(err) {
		return err
	} else if err != nil {
		klog.Infof("Failed to retrieve %s %s. err: %s", gvk.Kind, key.Name, err)
		return err
	}

	// Copy the value of the item in the cache to the returned value
	objVal := reflect.ValueOf(obj)
	itemVal := reflect.ValueOf(result)

	if !objVal.Type().AssignableTo(objVal.Type()) {
		return fmt.Errorf("cache had type %s, but %s was asked for", itemVal.Type(), objVal.Type())
	}
	reflect.Indirect(objVal).Set(reflect.Indirect(itemVal))
	obj.GetObjectKind().SetGroupVersionKind(gvk)

	return nil
}

// List lists items out of the indexer and writes them to list
func (cc filteredCache) List(ctx context.Context, list runtime.Object, opts ...cr_client.ListOption) error {
	gvk, err := apiutil.GVKForObject(list, cc.Scheme)
	if err != nil {
		return err
	}
	if informer, ok := cc.informerMap[gvk]; ok {
		// Construct filter
		var objList []interface{}

		listOpts := cr_client.ListOptions{}
		listOpts.ApplyOptions(opts)

		objList = informer.GetStore().List()

		var labelSel labels.Selector
		if listOpts.LabelSelector != nil {
			labelSel = listOpts.LabelSelector
		} else {
			return cc.fallback.List(ctx, list, opts...)
		}

		runtimeObjList := make([]runtime.Object, 0, len(objList))
		for _, item := range objList {
			obj, isObj := item.(runtime.Object)
			if !isObj {
				return fmt.Errorf("cache contained %T, which is not an Object", obj)
			}
			meta, err := apimeta.Accessor(obj)
			if err != nil {
				return err
			}

			if listOpts.Namespace != "" && listOpts.Namespace != meta.GetNamespace() {
				continue
			}

			if labelSel != nil {
				lbls := labels.Set(meta.GetLabels())
				if !labelSel.Matches(lbls) {
					continue
				}
			}

			outObj := obj.DeepCopyObject()
			outObj.GetObjectKind().SetGroupVersionKind(schema.GroupVersionKind{Group: gvk.Group, Version: gvk.Version, Kind: gvk.Kind[:len(gvk.Kind)-4]})
			runtimeObjList = append(runtimeObjList, outObj)
		}
		return apimeta.SetList(list, runtimeObjList)
	}

	// Passthrough
	return cc.fallback.List(ctx, list, opts...)
}

func (cc filteredCache) GetInformer(ctx context.Context, obj runtime.Object) (cr_cache.Informer, error) {
	gvk, err := apiutil.GVKForObject(obj, cc.Scheme)
	if err != nil {
		return nil, err
	}

	if informer, ok := cc.informerMap[gvk]; ok {
		return informer, nil
	}
	// Passthrough
	return cc.fallback.GetInformer(ctx, obj)
}

func (cc filteredCache) GetInformerForKind(ctx context.Context, gvk schema.GroupVersionKind) (cr_cache.Informer, error) {
	if informer, ok := cc.informerMap[gvk]; ok {
		return informer, nil
	}
	// Passthrough
	return cc.fallback.GetInformerForKind(ctx, gvk)
}

func (cc filteredCache) Start(stopCh <-chan struct{}) error {
	klog.Info("Start")
	for _, informer := range cc.informerMap {
		go informer.Run(stopCh)
	}
	return cc.fallback.Start(stopCh)
}

func (cc filteredCache) WaitForCacheSync(stop <-chan struct{}) bool {
	// Wait for informer to sync
	klog.Info("Waiting for informers to sync")
	waiting := true
	for waiting {
		select {
		case <-stop:
			waiting = false
		case <-time.After(time.Second):
			for _, informer := range cc.informerMap {
				waiting = !informer.HasSynced() && waiting
			}
		}
	}
	// Wait for fallback cache to sync
	klog.Info("Waiting for fallback informer to sync")
	return cc.fallback.WaitForCacheSync(stop)
}

func (cc filteredCache) IndexField(ctx context.Context, obj runtime.Object, field string, extractValue cr_client.IndexerFunc) error {
	gvk, err := apiutil.GVKForObject(obj, cc.Scheme)
	if err != nil {
		return err
	}

	if _, ok := cc.informerMap[gvk]; ok {
		klog.Infof("IndexField for %s not supported", gvk.String())
		return ErrUnsupported
	}

	return cc.fallback.IndexField(ctx, obj, field, extractValue)
}

func kindToResource(kind string) string {
	return strings.ToLower(flect.Pluralize(kind))
}
