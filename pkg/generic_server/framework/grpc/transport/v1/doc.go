// Package v1 provides generic gRPC transport layer utilities.
//
// It defines the Decoder, Encoder and Handler contracts with generic type
// constraints for building type-safe unary gRPC handlers, together with the
// server-streaming utilities: MsgAssembler and StreamSendMsg (chunked-send
// primitives), and StreamHandler with its functional counterpart ServeStream
// (the "compute + chunked send" composition).
//
// Channel-based streaming transport (a channel-driven counterpart of StreamSendMsg
// and StreamHandler) is planned; see the TODO(channel-transport) markers.
package v1
