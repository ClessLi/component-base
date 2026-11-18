// Copyright 2024 ClessLi. All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

// Package encoder is the request-encoder layer of the gRPC client SDK example's
// transport. Following the task_scheduler client SDK layout, it exposes a
// Factory that hands out one typed grpcclientv1.Encoder per RPC, converting a
// domain request into the corresponding protobuf request message so the
// transport layer can inject it into the generic gRPC client builder.
package encoder

// Factory is the entry point of the encoder layer.
type Factory interface {
	Demo() DemoEncoders
}

// Compile-time check that factory satisfies the factory contract.
var _ Factory = factory{}

type factory struct{}

func (f factory) Demo() DemoEncoders {
	return new(demoEncoders)
}

// New creates the encoder Factory.
func New() Factory {
	return new(factory)
}
