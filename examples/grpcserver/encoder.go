// Copyright 2024 ClessLi. All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package grpcserver

import (
	"context"

	pbv1 "github.com/ClessLi/component-base/examples/grpcserver/api/protobuf-spec/grpcserverpb/v1"
)

// encodeEchoResponse encodes the domain response into the protobuf Echo response.
func encodeEchoResponse(ctx context.Context, message string) (*pbv1.EchoResponse, error) {
	return &pbv1.EchoResponse{Message: message}, nil
}

// encodeDownloadResponse encodes the full domain payload into a single protobuf
// Chunk. The transport layer then splits it into stream chunks via StreamSendMsg.
func encodeDownloadResponse(ctx context.Context, data []byte) (*pbv1.Chunk, error) {
	return &pbv1.Chunk{Data: data}, nil
}
