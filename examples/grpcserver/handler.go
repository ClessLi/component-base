// Copyright 2024 ClessLi. All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package grpcserver

import (
	"sync"

	pbv1 "github.com/ClessLi/component-base/examples/grpcserver/api/protobuf-spec/grpcserverpb/v1"
	transportv1 "github.com/ClessLi/component-base/pkg/generic_server/framework/grpc/transport/v1"
)

// DemoHandlers is the transport-layer handler set for the demo service.
// Each handler is a lazily-initialized singleton, mirroring bifrost's handler factory.
type DemoHandlers interface {
	Echo() transportv1.Handler[*pbv1.EchoRequest, *pbv1.EchoResponse]
	Download() transportv1.StreamHandler[*pbv1.DownloadRequest, *pbv1.Chunk]
}

type demoHandlers struct {
	eps       DemoEndpoints
	chunkSize int

	onceEcho        sync.Once
	onceDownload    sync.Once
	echoHandler     transportv1.Handler[*pbv1.EchoRequest, *pbv1.EchoResponse]
	downloadHandler transportv1.StreamHandler[*pbv1.DownloadRequest, *pbv1.Chunk]
}

// Compile-time check that demoHandlers satisfies the handler-set contract.
var _ DemoHandlers = (*demoHandlers)(nil)

// NewDemoHandlers creates the handler set from the endpoint layer. chunkSize is
// injected from the generic server config for the streaming handler.
func NewDemoHandlers(eps DemoEndpoints, chunkSize int) DemoHandlers {
	return &demoHandlers{eps: eps, chunkSize: chunkSize}
}

func (h *demoHandlers) Echo() transportv1.Handler[*pbv1.EchoRequest, *pbv1.EchoResponse] {
	h.onceEcho.Do(func() {
		h.echoHandler = transportv1.NewHandler(h.eps.Echo(), decodeEchoRequest, encodeEchoResponse)
	})
	return h.echoHandler
}

func (h *demoHandlers) Download() transportv1.StreamHandler[*pbv1.DownloadRequest, *pbv1.Chunk] {
	h.onceDownload.Do(func() {
		h.downloadHandler = transportv1.NewStreamHandler(
			h.eps.Download(), decodeDownloadRequest, encodeDownloadResponse,
			h.chunkSize,
			(*pbv1.Chunk).GetData,
			func(msg []byte) *pbv1.Chunk { return &pbv1.Chunk{Data: msg} },
		)
	})
	return h.downloadHandler
}
