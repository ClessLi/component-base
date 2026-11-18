// Copyright 2024 ClessLi. All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package encoder

import (
	"context"

	pbv1 "github.com/ClessLi/component-base/examples/grpcserver/api/protobuf-spec/grpcserverpb/v1"
	grpcclientv1 "github.com/ClessLi/component-base/pkg/client-sdk/grpc/v1"
)

// DemoEncoders exposes the typed request encoders of each Demo RPC, matching the
// generic gRPC client SDK contract. It converts a domain request into the
// corresponding protobuf request message.
type DemoEncoders interface {
	Echo() grpcclientv1.Encoder[string, *pbv1.EchoRequest]
}

// Compile-time checks documenting the SDK encoder contract for each RPC.
var (
	_ DemoEncoders                                    = demoEncoders{}
	_ grpcclientv1.Encoder[string, *pbv1.EchoRequest] = demoEncoders{}.Echo()
)

// demoEncoders implements DemoEncoders with the typed request encoders of each RPC.
type demoEncoders struct{}

// Echo converts a domain message into the EchoRequest protobuf message.
func (demoEncoders) Echo() grpcclientv1.Encoder[string, *pbv1.EchoRequest] {
	return func(_ context.Context, msg string) (*pbv1.EchoRequest, error) {
		return &pbv1.EchoRequest{Message: msg}, nil
	}
}
