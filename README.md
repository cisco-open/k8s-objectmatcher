![license](http://img.shields.io/badge/license-Apache%20v2-orange.svg)

# Kubernetes object matcher

K8S-ObjectMatcher is a Golang library which helps to match Kubernetes objects.

### Motivation

Here at Banzai Cloud we love and write lots of Kubernetes [operators](https://github.com/banzaicloud?utf8=✓&q=operator&type=&language=). While writing some complex operators as the [Istio](https://github.com/banzaicloud/istio-operator) , [Vault](https://github.com/banzaicloud/bank-vaults) or [Kafka](https://github.com/banzaicloud/kafka-operator) operator, we encountered a huge amount of **unnecessary Kubernetes object updates**. Most of the operators out there are using `reflect.DeepEquals` to match the given object's `Spec`. Unfortunately, this solution is not perfect because every Kubernetes object is amended with different default values while submitted. This library aims to provide finer object matching capabilities to avoid unnecessary updates and more observability on the client side.

### Legacy version deprecation notice

There is a legacy version of the lib, that is now deprecated and documented here: [docs/legacy.md](docs/legacy.md)

### How does it work?

The library uses the same method that `kubectl apply` does under the hood to calculate a patch using the [three way merge](http://www.drdobbs.com/tools/three-way-merging-a-look-under-the-hood/240164902) method.
However for this to work properly we need to keep track of the last applied version of our object, let's call it the `original`. Unfortunately Kubernetes does
not keep track of our previously submitted object versions, but we can put it into an annotation like `kubectl apply` does. 
Next time we query the `current` state of the object from the API Server we can extract the `original` version from the annotation.

Once we have the the `original`, the `current` and our new `modified` object in place the library will take care of the rest.

#### Example steps demonstrated on a v1.Service object

Create a new object, annotate it, then submit normally
```go
original := &v1.Service{
  ...
}

patch.DefaultAnnotator.SetLastAppliedAnnotation(original)

client.CoreV1().Services(original.GetNamespace()).Create(original)
```

Next time we check the diff and set the last applied annotation in case we have to update
```go
modified := &v1.Service{
  ...
}

current, err := client.CoreV1().Services(modified.GetNamespace()).Get(modified.GetName(), metav1.Getoptions{})

patchResult, err := patch.DefaultPatchMaker.Calculate(current, modified)
if err != nil {
  return err
}

if !patchResult.IsEmpty() {
  patch.DefaultAnnotator.SetLastAppliedAnnotation(modified)
  client.CoreV1().Services(modified.GetNamespace()).Update(modified)
}

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
