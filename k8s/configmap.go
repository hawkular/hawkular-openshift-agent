/*
   Copyright 2016 Red Hat, Inc. and/or its affiliates
   and other contributors.

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

package k8s

import (
	"fmt"
)

type ConfigMap struct {
	Namespace string
	Name      string
	Entry     *ConfigMapEntry
}

func NewConfigMap(namespace string, name string, cme *ConfigMapEntry) (cm *ConfigMap) {
	cm = &ConfigMap{
		Namespace: namespace,
		Name:      name,
		Entry:     cme,
	}
	return
}

func (cm *ConfigMap) String() string {
	return fmt.Sprintf("ConfigMap[%v:%v]: %v", cm.Namespace, cm.Name, cm.Entry)
}
