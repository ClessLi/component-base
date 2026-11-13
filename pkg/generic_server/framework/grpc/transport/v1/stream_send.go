package v1

import (
	"github.com/ClessLi/component-base/pkg/errors"
	"google.golang.org/grpc"
)

// MsgAssembler assembles a chunk of bytes into a protobuf message.
// ProtoMsg: protobuf message type carried by the stream
type MsgAssembler[ProtoMsg any] func(msg []byte) ProtoMsg

// StreamSendMsg sends data over a gRPC server stream in chunks of at most chunksize bytes.
// Each chunk is assembled into a protobuf message by the assembler before sending.
// It is used for chunked unary-over-stream responses and server-side streaming pushes.
//
// Send errors (including io.EOF) are returned unwrapped so that callers can inspect
// them with errors.Is. Empty data sends no message.
func StreamSendMsg[ProtoMsg any](stream grpc.ServerStream, data []byte, chunksize int, assembler MsgAssembler[ProtoMsg]) error {
	if chunksize <= 0 {
		return errors.Errorf("invalid chunk size: %d, must be greater than 0", chunksize)
	}
	if assembler == nil {
		return errors.New("message assembler must not be nil")
	}

	for i := 0; i < len(data); i += chunksize {
		// Stop early if the stream context is done (client cancelled or deadline exceeded).
		if err := stream.Context().Err(); err != nil {
			return err
		}

		end := i + chunksize
		if end > len(data) {
			end = len(data)
		}

		if err := stream.SendMsg(assembler(data[i:end])); err != nil {
			return err
		}
	}

	return nil
}
