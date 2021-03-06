/*
Copyright 2019 The Codefresh Authors.

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

package plugins

import (
	"fmt"

	"github.com/codefresh-io/venona/venonactl/pkg/logger"
	"github.com/codefresh-io/venona/venonactl/pkg/obj/kubeobj"
	templates "github.com/codefresh-io/venona/venonactl/pkg/templates/kubernetes"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// enginePlugin installs assets on Kubernetes Dind runtimectl Env
type enginePlugin struct {
	logger logger.Logger
}

const (
	engineFilesPattern = ".*.engine.yaml"
)

// Install venona agent
func (u *enginePlugin) Install(opt *InstallOptions, v Values) (Values, error) {
	cs, err := opt.KubeBuilder.BuildClient()
	if err != nil {
		u.logger.Error(fmt.Sprintf("Cannot create kubernetes clientset: %v ", err))
		return nil, err
	}
	return v, install(&installOptions{
		logger:         u.logger,
		templates:      templates.TemplatesMap(),
		templateValues: v,
		kubeClientSet:  cs,
		namespace:      opt.ClusterNamespace,
		matchPattern:   engineFilesPattern,
		dryRun:         opt.DryRun,
		operatorType:   EnginePluginType,
	})
}

// Status of runtimectl environment
func (u *enginePlugin) Status(statusOpt *StatusOptions, v Values) ([][]string, error) {
	cs, err := statusOpt.KubeBuilder.BuildClient()
	if err != nil {
		u.logger.Error(fmt.Sprintf("Cannot create kubernetes clientset: %v ", err))
		return nil, err
	}
	opt := &statusOptions{
		logger:         u.logger,
		templates:      templates.TemplatesMap(),
		templateValues: v,
		kubeClientSet:  cs,
		namespace:      statusOpt.ClusterNamespace,
		matchPattern:   engineFilesPattern,
		operatorType:   EnginePluginType,
	}
	return status(opt)
}

func (u *enginePlugin) Delete(deleteOpt *DeleteOptions, v Values) error {
	cs, err := deleteOpt.KubeBuilder.BuildClient()
	if err != nil {
		u.logger.Error(fmt.Sprintf("Cannot create kubernetes clientset: %v ", err))
		return nil
	}
	opt := &deleteOptions{
		logger:         u.logger,
		templates:      templates.TemplatesMap(),
		templateValues: v,
		kubeClientSet:  cs,
		namespace:      deleteOpt.ClusterNamespace,
		matchPattern:   engineFilesPattern,
		operatorType:   EnginePluginType,
	}
	return delete(opt)
}

func (u *enginePlugin) Upgrade(opt *UpgradeOptions, v Values) (Values, error) {

	// replace of sa creates new secert with sa creds
	// avoid it till patch fully implemented
	var skipUpgradeFor = map[string]interface{}{
		"service-account.venona.yaml": nil,
	}

	var err error

	kubeClientset, err := opt.KubeBuilder.BuildClient()
	if err != nil {
		u.logger.Error(fmt.Sprintf("Cannot create kubernetes clientset: %v ", err))
		return nil, err
	}

	// special case when we need to get the token from the remote to no regenrate it
	// whole flow should be more like kubectl apply that build a patch
	// based on remote object and candidate object

	secret, err := kubeClientset.CoreV1().Secrets(opt.ClusterNamespace).Get(opt.Name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	token := secret.Data["codefresh.token"]
	v["AgentToken"] = string(token)

	kubeObjects, err := getKubeObjectsFromTempalte(v, engineFilesPattern, u.logger)
	if err != nil {
		return nil, err
	}

	for fileName, local := range kubeObjects {
		if _, ok := skipUpgradeFor[fileName]; ok {
			u.logger.Debug(fmt.Sprintf("Skipping upgrade of %s: should be ignored", fileName))
			continue
		}

		_, _, err := kubeobj.ReplaceObject(kubeClientset, local, opt.ClusterNamespace)
		if err != nil {
			return nil, err
		}
	}

	return v, nil
}
