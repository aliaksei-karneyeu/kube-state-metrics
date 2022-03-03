/*
Copyright 2016 The Kubernetes Authors All rights reserved.

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

package store

import (
	"context"

	"k8s.io/kube-state-metrics/v2/pkg/metric"
	generator "k8s.io/kube-state-metrics/v2/pkg/metric_generator"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)

var (
	descPodLabelsDefaultLabels = []string{"namespace", "pod", "uid"}
	podStatusReasons           = []string{"Evicted", "NodeAffinity", "NodeLost", "Shutdown", "UnexpectedAdmissionError"}
)

func podMetricFamilies(allowAnnotationsList, allowLabelsList []string) []generator.FamilyGenerator {
	return []generator.FamilyGenerator{
		createPodStatusReadyTimeFamilyGenerator(),
		createPodStatusContainersReadyTimeFamilyGenerator(),
	}
}

func createPodStatusContainersReadyTimeFamilyGenerator() generator.FamilyGenerator {
	return *generator.NewFamilyGenerator(
		"kube_pod_status_containers_ready_time",
		"Readiness achieved time in unix timestamp for a pod containers.",
		metric.Gauge,
		"",
		wrapPodFunc(func(p *v1.Pod) *metric.Family {
			ms := []*metric.Metric{}

			for _, c := range p.Status.Conditions {
				if c.Type == v1.ContainersReady {
					ms = append(ms, &metric.Metric{
						LabelKeys:   []string{},
						LabelValues: []string{},
						Value:       float64((c.LastTransitionTime).Unix()),
					})
				}
			}

			return &metric.Family{
				Metrics: ms,
			}
		}),
	)
}

func createPodStatusReadyTimeFamilyGenerator() generator.FamilyGenerator {
	return *generator.NewFamilyGenerator(
		"kube_pod_status_ready_time",
		"Readiness achieved time in unix timestamp for a pod.",
		metric.Gauge,
		"",
		wrapPodFunc(func(p *v1.Pod) *metric.Family {
			ms := []*metric.Metric{}

			for _, c := range p.Status.Conditions {
				if c.Type == v1.PodReady {
					ms = append(ms, &metric.Metric{
						LabelKeys:   []string{},
						LabelValues: []string{},
						Value:       float64((c.LastTransitionTime).Unix()),
					})
				}
			}

			return &metric.Family{
				Metrics: ms,
			}
		}),
	)
}

func wrapPodFunc(f func(*v1.Pod) *metric.Family) func(interface{}) *metric.Family {
	return func(obj interface{}) *metric.Family {
		pod := obj.(*v1.Pod)

		metricFamily := f(pod)

		for _, m := range metricFamily.Metrics {
			m.LabelKeys, m.LabelValues = mergeKeyValues(descPodLabelsDefaultLabels, []string{pod.Namespace, pod.Name, string(pod.UID)}, m.LabelKeys, m.LabelValues)
		}

		return metricFamily
	}
}

func createPodListWatch(kubeClient clientset.Interface, ns string, fieldSelector string) cache.ListerWatcher {
	return &cache.ListWatch{
		ListFunc: func(opts metav1.ListOptions) (runtime.Object, error) {
			opts.FieldSelector = fieldSelector
			return kubeClient.CoreV1().Pods(ns).List(context.TODO(), opts)
		},
		WatchFunc: func(opts metav1.ListOptions) (watch.Interface, error) {
			opts.FieldSelector = fieldSelector
			return kubeClient.CoreV1().Pods(ns).Watch(context.TODO(), opts)
		},
	}
}
