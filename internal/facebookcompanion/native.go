package facebookcompanion

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	"ContentBlueprint/internal/workbench"
)

const (
	NativeHostName        = "com.contentblueprint.facebook"
	NativeProtocolVersion = "1.0"
	MaxNativeMessageBytes = 1_048_576
	// Generated from the development manifest key. Keeping both values fixed is
	// required because Chrome Native Messaging does not allow wildcard origins.
	ExtensionID            = "ppncejmpiekmkepaeccdnpnpgdcfafje"
	AllowedExtensionOrigin = "chrome-extension://" + ExtensionID + "/"
)

type NativeRequest struct {
	Action        string `json:"action"`
	Brief         *Brief `json:"brief,omitempty"`
	BriefRevision string `json:"briefRevision,omitempty"`
}

type NativeResponse struct {
	OK              bool                    `json:"ok"`
	Action          string                  `json:"action,omitempty"`
	ProtocolVersion string                  `json:"protocolVersion,omitempty"`
	Found           bool                    `json:"found,omitempty"`
	Stale           bool                    `json:"stale,omitempty"`
	BriefRevision   string                  `json:"briefRevision,omitempty"`
	UpdatedAt       string                  `json:"updatedAt,omitempty"`
	Snapshot        *PackSnapshot           `json:"snapshot,omitempty"`
	GrowthSnapshot  *workbench.PackSnapshot `json:"growthSnapshot,omitempty"`
	ErrorCode       string                  `json:"errorCode,omitempty"`
	Message         string                  `json:"message,omitempty"`
}

func ValidateNativeOrigin(origin string) error {
	if origin != AllowedExtensionOrigin {
		return fmt.Errorf("native messaging caller is not the Content Blueprint extension")
	}
	return nil
}

// RunNativeHost serves Chrome's length-prefixed JSON protocol. Chrome starts a
// new process for sendNativeMessage and may keep one alive for connectNative,
// so the function safely supports either one or multiple requests.
func RunNativeHost(ctx context.Context, reader io.Reader, writer io.Writer, store *Store) error {
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		payload, err := readNativeFrame(reader)
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return err
		}
		response := HandleNativeRequest(payload, store)
		if err := writeNativeFrame(writer, response); err != nil {
			return err
		}
	}
}

func HandleNativeRequest(payload []byte, store *Store) NativeResponse {
	if store == nil {
		return nativeError("INTERNAL_ERROR", "companion storage is unavailable")
	}
	if len(payload) == 0 || len(payload) > MaxNativeMessageBytes {
		return nativeError("INVALID_MESSAGE", "native message is empty or too large")
	}
	var request NativeRequest
	decoder := json.NewDecoder(bytes.NewReader(payload))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&request); err != nil {
		return nativeError("INVALID_MESSAGE", "native message must match the companion protocol")
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		return nativeError("INVALID_MESSAGE", "native message contains trailing data")
	}
	request.Action = strings.TrimSpace(request.Action)
	switch request.Action {
	case "health":
		return NativeResponse{
			OK:              true,
			Action:          request.Action,
			ProtocolVersion: NativeProtocolVersion,
		}
	case "saveBrief":
		if request.Brief == nil {
			return nativeError("INVALID_BRIEF", "brief is required")
		}
		snapshot, err := store.SaveBrief(*request.Brief)
		if err != nil {
			return nativeError("INVALID_BRIEF", err.Error())
		}
		return NativeResponse{
			OK:              true,
			Action:          request.Action,
			ProtocolVersion: NativeProtocolVersion,
			BriefRevision:   snapshot.BriefRevision,
			UpdatedAt:       snapshot.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
		}
	case "getLatestPack":
		pack, err := store.LoadPack()
		if errors.Is(err, ErrNotFound) {
			return NativeResponse{
				OK:              true,
				Action:          request.Action,
				ProtocolVersion: NativeProtocolVersion,
				Found:           false,
			}
		}
		if err != nil {
			return nativeError("STATE_ERROR", "the saved Content Pack could not be read")
		}
		brief, briefErr := store.LoadBrief()
		if briefErr != nil && !errors.Is(briefErr, ErrNotFound) {
			return nativeError("STATE_ERROR", "the saved brief could not be read")
		}
		expectedRevision := strings.TrimSpace(request.BriefRevision)
		stale := errors.Is(briefErr, ErrNotFound) || brief.BriefRevision != pack.BriefRevision
		if expectedRevision != "" && expectedRevision != pack.BriefRevision {
			stale = true
		}
		return NativeResponse{
			OK:              true,
			Action:          request.Action,
			ProtocolVersion: NativeProtocolVersion,
			Found:           true,
			Stale:           stale,
			BriefRevision:   pack.BriefRevision,
			UpdatedAt:       pack.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
			Snapshot:        &pack,
		}
	case "getLatestGrowthPack":
		growthStore, err := workbench.NewStore("")
		if err != nil {
			return nativeError("STATE_ERROR", "Growth Workbench storage is unavailable")
		}
		pack, err := growthStore.LoadPack()
		if errors.Is(err, workbench.ErrNotFound) {
			return NativeResponse{OK: true, Action: request.Action, ProtocolVersion: NativeProtocolVersion, Found: false}
		}
		if err != nil {
			return nativeError("STATE_ERROR", "the saved Growth Pack could not be read")
		}
		brief, briefErr := growthStore.LoadBrief()
		if briefErr != nil && !errors.Is(briefErr, workbench.ErrNotFound) {
			return nativeError("STATE_ERROR", "the saved Growth Brief could not be read")
		}
		expectedRevision := strings.TrimSpace(request.BriefRevision)
		stale := errors.Is(briefErr, workbench.ErrNotFound) || brief.BriefRevision != pack.BriefRevision
		if expectedRevision != "" && expectedRevision != pack.BriefRevision {
			stale = true
		}
		response := NativeResponse{
			OK: true, Action: request.Action, ProtocolVersion: NativeProtocolVersion,
			Found: true, Stale: stale, BriefRevision: pack.BriefRevision,
			UpdatedAt: pack.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"), GrowthSnapshot: &pack,
		}
		if encoded, encodeErr := json.Marshal(response); encodeErr != nil || len(encoded) > MaxNativeMessageBytes {
			return nativeError("STATE_TOO_LARGE", "the saved Growth Pack is too large for native messaging")
		}
		return response
	default:
		return nativeError("UNKNOWN_ACTION", "unsupported companion action")
	}
}

func nativeError(code, message string) NativeResponse {
	message = strings.TrimSpace(message)
	if len(message) > 600 {
		message = message[:600]
	}
	return NativeResponse{
		OK:              false,
		ProtocolVersion: NativeProtocolVersion,
		ErrorCode:       code,
		Message:         message,
	}
}

func readNativeFrame(reader io.Reader) ([]byte, error) {
	var lengthBytes [4]byte
	if _, err := io.ReadFull(reader, lengthBytes[:]); err != nil {
		if errors.Is(err, io.EOF) {
			return nil, io.EOF
		}
		return nil, fmt.Errorf("read native message length: %w", err)
	}
	length := binary.LittleEndian.Uint32(lengthBytes[:])
	if length == 0 || length > MaxNativeMessageBytes {
		return nil, fmt.Errorf("native message length %d is outside the supported range", length)
	}
	payload := make([]byte, int(length))
	if _, err := io.ReadFull(reader, payload); err != nil {
		return nil, fmt.Errorf("read native message: %w", err)
	}
	return payload, nil
}

func writeNativeFrame(writer io.Writer, response NativeResponse) error {
	payload, err := json.Marshal(response)
	if err != nil {
		return fmt.Errorf("encode native response: %w", err)
	}
	if len(payload) == 0 || len(payload) > MaxNativeMessageBytes {
		return fmt.Errorf("native response exceeds Chrome's one-megabyte limit")
	}
	var lengthBytes [4]byte
	binary.LittleEndian.PutUint32(lengthBytes[:], uint32(len(payload)))
	if err := writeAll(writer, lengthBytes[:]); err != nil {
		return fmt.Errorf("write native response length: %w", err)
	}
	if err := writeAll(writer, payload); err != nil {
		return fmt.Errorf("write native response: %w", err)
	}
	return nil
}

func writeAll(writer io.Writer, payload []byte) error {
	for len(payload) > 0 {
		written, err := writer.Write(payload)
		if err != nil {
			return err
		}
		if written == 0 {
			return io.ErrShortWrite
		}
		payload = payload[written:]
	}
	return nil
}
