/*
Copyright 2023 K8sGPT Contributors.

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

package v1alpha1

import (
	"fmt"
	"log"
	"os"
	"runtime"
	"strings"

	"gopkg.in/yaml.v2"
)

func (r *K8sGPT) LoadResource(path string) *K8sGPT {
	// Path to this file
	_, filename, _, _ := runtime.Caller(0)
	pSlice := strings.Split(filename, "/")
	projectRoot := strings.Join(pSlice[:len(pSlice)-3], "/")
	yamlFile, err := os.ReadFile(projectRoot + "/" + path)
	if err != nil {
		log.Printf("yamlFile.Get err   #%v ", err)
	}
	err = yaml.Unmarshal(yamlFile, r)
	if err != nil {
		log.Fatalf("Unmarshal: %v", err)
	}

	fmt.Println(r.Name)

	return r
}

func (r *K8sGPT) GetValidResource() *K8sGPT {
	return r.LoadResource("./config/samples/valid_k8sgpt.yaml")
}

func GetValidProjectResource(name, namespace string) K8sGPT {
	k8sgpt := K8sGPT{}
	k8sgpt.GetValidResource()
	k8sgpt.Name = name
	k8sgpt.Namespace = namespace

	return k8sgpt
}
