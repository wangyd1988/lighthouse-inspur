/*
SPDX-License-Identifier: Apache-2.0

Copyright Contributors to the Submariner project.

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

package resolver

import (
	"github.com/pkg/errors"
	"github.com/wangyd1988/admiral-inspur/pkg/log"
	"github.com/wangyd1988/admiral-inspur/pkg/watcher"
	"github.com/wangyd1988/lighthouse-inspur/coredns/constants"
	discovery "k8s.io/api/discovery/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	mcsv1a1 "sigs.k8s.io/mcs-api/pkg/apis/v1alpha1"
)

var logger = log.Logger{Logger: logf.Log.WithName("Resolver")}

type controller struct {
	resolver *Interface
	stopCh   chan struct{}
}

func NewController(r *Interface) *controller {
	return &controller{
		resolver: r,
		stopCh:   make(chan struct{}),
	}
}

func (c *controller) Start(config watcher.Config) error {
	logger.Infof("Starting Resolver Controller")

	config.ResourceConfigs = []watcher.ResourceConfig{
		{
			Name:                "EndpointSlice watcher",
			ResourceType:        &discovery.EndpointSlice{},
			SourceNamespace:     metav1.NamespaceAll,
			SourceLabelSelector: labels.Set(map[string]string{discovery.LabelManagedBy: constants.LabelValueManagedBy}).String(),
			Handler: watcher.EventHandlerFuncs{
				OnCreateFunc: c.onEndpointSliceCreateOrUpdate,
				OnUpdateFunc: c.onEndpointSliceCreateOrUpdate,
				OnDeleteFunc: c.onEndpointSliceDelete,
			},
		},
		{
			Name:            "ServiceImport watcher",
			ResourceType:    &mcsv1a1.ServiceImport{},
			SourceNamespace: metav1.NamespaceAll,
			Handler: watcher.EventHandlerFuncs{
				OnCreateFunc: c.onServiceImportCreateOrUpdate,
				OnUpdateFunc: c.onServiceImportCreateOrUpdate,
				OnDeleteFunc: c.onServiceImportDelete,
			},
		},
	}

	resourceWatcher, err := watcher.New(&config)
	if err != nil {
		return errors.Wrap(err, "error creating the resource watcher")
	}

	err = resourceWatcher.Start(c.stopCh)
	if err != nil {
		return errors.Wrap(err, "error starting the resource watcher")
	}

	return nil
}

func (c *controller) Stop() {
	close(c.stopCh)

	logger.Infof("Resolver Controller stopped")
}

func (c *controller) onEndpointSliceCreateOrUpdate(obj runtime.Object, _ int) bool {
	return c.resolver.PutEndpointSlice(obj.(*discovery.EndpointSlice))
}

func (c *controller) onEndpointSliceDelete(obj runtime.Object, _ int) bool {
	c.resolver.RemoveEndpointSlice(obj.(*discovery.EndpointSlice))
	return false
}

func (c *controller) onServiceImportCreateOrUpdate(obj runtime.Object, _ int) bool {
	c.resolver.PutServiceImport(obj.(*mcsv1a1.ServiceImport))
	return false
}

func (c *controller) onServiceImportDelete(obj runtime.Object, _ int) bool {
	c.resolver.RemoveServiceImport(obj.(*mcsv1a1.ServiceImport))
	return false
}
