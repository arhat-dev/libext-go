package extruntime

import (
	"context"

	"arhat.dev/aranya-proto/aranyagopb/runtimepb"
)

func (h *Handler) handleImageList(ctx context.Context, sid uint64, payload []byte) (*runtimepb.Packet, error) {
	opts := new(runtimepb.ImageListCmd)
	err := opts.Unmarshal(payload)
	if err != nil {
		return nil, err
	}

	msg, err := h.impl.ListImages(ctx, opts)
	if err != nil {
		return nil, err
	}

	data, err := (&runtimepb.ImageStatusListMsg{Images: msg}).Marshal()
	if err != nil {
		return nil, err
	}

	return &runtimepb.Packet{
		Kind:    runtimepb.MSG_IMAGE_STATUS_LIST,
		Payload: data,
	}, nil
}

func (h *Handler) handleImageEnsure(ctx context.Context, sid uint64, payload []byte) (*runtimepb.Packet, error) {
	opts := new(runtimepb.ImageEnsureCmd)
	err := opts.Unmarshal(payload)
	if err != nil {
		return nil, err
	}

	msg, err := h.impl.EnsureImages(ctx, opts)
	if err != nil {
		return nil, err
	}

	data, err := (&runtimepb.ImageStatusListMsg{Images: msg}).Marshal()
	if err != nil {
		return nil, err
	}

	return &runtimepb.Packet{
		Kind:    runtimepb.MSG_IMAGE_STATUS_LIST,
		Payload: data,
	}, nil
}

func (h *Handler) handleImageDelete(ctx context.Context, sid uint64, payload []byte) (*runtimepb.Packet, error) {
	opts := new(runtimepb.ImageDeleteCmd)
	err := opts.Unmarshal(payload)
	if err != nil {
		return nil, err
	}

	msg, err := h.impl.DeleteImages(ctx, opts)
	if err != nil {
		return nil, err
	}

	data, err := (&runtimepb.ImageStatusListMsg{Images: msg}).Marshal()
	if err != nil {
		return nil, err
	}

	return &runtimepb.Packet{
		Kind:    runtimepb.MSG_IMAGE_STATUS_LIST,
		Payload: data,
	}, nil
}
