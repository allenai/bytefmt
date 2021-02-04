# bytefmt

Package bytefmt is a utility to parse, format, and manipulate byte quantities.
This package emphasizes accuracy and adherence to standards over performance and
implements both binary and decimal International System of Units (SI) conventions.

This package is inspired by Kubernetes'
[`resource.Quantity`](https://pkg.go.dev/k8s.io/apimachinery@v0.20.2/pkg/api/resource).
It carries no dependencies and a simplified interface focused strictly on byte
quantities.

## Alternatives

- [`code.cloudfoundry.org/bytefmt`](https://github.com/cloudfoundry/bytefmt):
  Simple, fast, and popular. This package assumes Binary SI convention and is
  limited to float64 precision.

- [`k8s.io/apimachinery/pkg/api/resource`](https://github.com/kubernetes/apimachinery):
  `resource.Quantity` provides exact precision, multiple standards, and
  arbitrarily large values. Being part of the larger Kubernetes API, it pulls in
  a large set of dependencies

- [`github.com/inhies/go-bytesize`](https://github.com/inhies/go-bytesize):
  `bytesize.ByteSize` Supports parsing and formatting full-word units such as
  "gigabyte". This package assumes Binary SI convention, but does not support SI
  prefixes: Ki, Mi, Gi, .... It is limited to int64 values.
