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

package filteredcache

import (
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	certificatesv1beta1 "k8s.io/api/certificates/v1beta1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	storagev1 "k8s.io/api/storage/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes"
	toolscache "k8s.io/client-go/tools/cache"
)

func getClientForGVK(gvk schema.GroupVersionKind, k8sClient *kubernetes.Clientset) toolscache.Getter {
	switch gvk.GroupVersion() {
	case corev1.SchemeGroupVersion:
		return k8sClient.CoreV1().RESTClient()
	case appsv1.SchemeGroupVersion:
		return k8sClient.AppsV1().RESTClient()
	case batchv1.SchemeGroupVersion:
		return k8sClient.BatchV1().RESTClient()
	case networkingv1.SchemeGroupVersion:
		return k8sClient.NetworkingV1().RESTClient()
	case rbacv1.SchemeGroupVersion:
		return k8sClient.RbacV1().RESTClient()
	case storagev1.SchemeGroupVersion:
		return k8sClient.StorageV1().RESTClient()
	case certificatesv1beta1.SchemeGroupVersion:
		return k8sClient.CertificatesV1beta1().RESTClient()
	}
	return nil
}
