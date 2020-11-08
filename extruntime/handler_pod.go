package extruntime

import (
	"context"

	"arhat.dev/aranya-proto/aranyagopb/runtimepb"
)

func (h *Handler) handlePodList(ctx context.Context, sid uint64, payload []byte) (*runtimepb.Packet, error) {
	opts := new(runtimepb.PodListCmd)
	err := opts.Unmarshal(payload)
	if err != nil {
		return nil, err
	}

	msg, err := h.impl.ListPods(ctx, opts)
	if err != nil {
		return nil, err
	}

	data, err := (&runtimepb.PodStatusListMsg{Pods: msg}).Marshal()
	if err != nil {
		return nil, err
	}

	return &runtimepb.Packet{
		Kind:    runtimepb.MSG_POD_STATUS_LIST,
		Payload: data,
	}, nil
}

func (h *Handler) handlePodEnsure(ctx context.Context, sid uint64, payload []byte) (*runtimepb.Packet, error) {
	opts := new(runtimepb.PodEnsureCmd)
	err := opts.Unmarshal(payload)
	if err != nil {
		return nil, err
	}

	msg, err := h.impl.EnsurePod(ctx, opts)
	if err != nil {
		return nil, err
	}

	data, err := msg.Marshal()
	if err != nil {
		return nil, err
	}

	return &runtimepb.Packet{
		Kind:    runtimepb.MSG_POD_STATUS,
		Payload: data,
	}, nil
}

func (h *Handler) handlePodDelete(ctx context.Context, sid uint64, payload []byte) (*runtimepb.Packet, error) {
	opts := new(runtimepb.PodDeleteCmd)
	err := opts.Unmarshal(payload)
	if err != nil {
		return nil, err
	}

	msg, err := h.impl.DeletePod(ctx, opts)
	if err != nil {
		return nil, err
	}

	data, err := msg.Marshal()
	if err != nil {
		return nil, err
	}

	return &runtimepb.Packet{
		Kind:    runtimepb.MSG_POD_STATUS,
		Payload: data,
	}, nil
}
