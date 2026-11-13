// Copyright 2024 ClessLi. All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package grpcserver

import (
	"context"

	pbv1 "github.com/ClessLi/component-base/examples/grpcserver/api/protobuf-spec/grpcserverpb/v1"
)

// decodeEchoRequest decodes the protobuf Echo request into the domain request.
func decodeEchoRequest(ctx context.Context, r *pbv1.EchoRequest) (string, error) {
	return r.GetMessage(), nil
}

// decodeDownloadRequest decodes the protobuf Download request into the domain request.
func decodeDownloadRequest(ctx context.Context, r *pbv1.DownloadRequest) (string, error) {
	return r.GetKey(), nil
}
