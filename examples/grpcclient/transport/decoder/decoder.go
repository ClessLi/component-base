// Copyright 2024 ClessLi. All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

// Package decoder is the response-decoder layer of the gRPC client SDK example's
// transport. Following the task_scheduler client SDK layout, it exposes a
// Factory that hands out the typed grpcclientv1 decoders per RPC: a unary
// Decoder for Echo, and the streaming MsgExtractor/StreamDecoder for Download,
// converting protobuf messages back into domain results.
package decoder

// Factory is the entry point of the decoder layer.
type Factory interface {
	Demo() DemoDecoders
}

// Compile-time check that factory satisfies the factory contract.
var _ Factory = factory{}

type factory struct{}

func (f factory) Demo() DemoDecoders {
	return new(demoDecoders)
}

// New creates the decoder Factory.
func New() Factory {
	return new(factory)
}
