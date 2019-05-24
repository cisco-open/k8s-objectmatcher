![license](http://img.shields.io/badge/license-Apache%20v2-orange.svg)

# Kubernetes object matcher

K8S-ObjectMatcher is a Golang library which helps to match Kubernetes objects.

### Motivation

Here at Banzai Cloud we love and write lots of Kubernetes [operators](https://github.com/banzaicloud?utf8=✓&q=operator&type=&language=). While writing some complex operators as the [Istio](https://github.com/banzaicloud/istio-operator) , [Vault](https://github.com/banzaicloud/bank-vaults) or [Kafka](https://github.com/banzaicloud/kafka-operator) operator, we encountered a huge amount of **unnecessary Kubernetes object updates**. Most of the operators out there are using `reflect.DeepEquals` to match the given object's `Spec`. Unfortunately, this solution is not perfect because every Kubernetes object is amended with different default values while submitted. This library aims to provide finer object matching capabilities to avoid unnecessary updates and more observability on the client side.

### Supported Objects

- ClusterRole
- ClusterRoleBindins
- ConfigMap
- CustomResourceDefinition
- DaemonSet
- Deployment
- HorizontalPodAutoScaler
- MutatingWebhook
- Role
- RoleBinding
- Pod
- PersistentVolumeClaim
- Service
- ServiceAccount
- PodDisruptionBudget
- Unstructured

### How to use it

```
objectMatcher := objectmatch.New(logf.NewDelegatingLogger(logf.NullLogger{}))
objectMatcher.Match(e.ObjectOld, e.ObjectNew)
```

## Contributing

If you find this project useful here's how you can help:

- Send a pull request with your new features and bug fixes
- Help new users with issues they may encounter
- Support the development of this project and star this repo!

## License

Copyright (c) 2017-2019 [Banzai Cloud, Inc.](https://banzaicloud.com)

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

[http://www.apache.org/licenses/LICENSE-2.0](http://www.apache.org/licenses/LICENSE-2.0)

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
