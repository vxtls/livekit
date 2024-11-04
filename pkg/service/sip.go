// Copyright 2023 LiveKit, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package service

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"time"

	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/livekit/protocol/livekit"
	"github.com/livekit/protocol/logger"
	"github.com/livekit/protocol/rpc"
	"github.com/livekit/protocol/sip"
	"github.com/livekit/protocol/utils"
	"github.com/livekit/protocol/utils/guid"
	"github.com/livekit/psrpc"

	"github.com/livekit/livekit-server/pkg/config"
	"github.com/livekit/livekit-server/pkg/telemetry"
)

type SIPService struct {
	conf        *config.SIPConfig
	nodeID      livekit.NodeID
	bus         psrpc.MessageBus
	psrpcClient rpc.SIPClient
	store       SIPStore
	roomService livekit.RoomService
}

func NewSIPService(
	conf *config.SIPConfig,
	nodeID livekit.NodeID,
	bus psrpc.MessageBus,
	psrpcClient rpc.SIPClient,
	store SIPStore,
	rs livekit.RoomService,
	ts telemetry.TelemetryService,
) *SIPService {
	return &SIPService{
		conf:        conf,
		nodeID:      nodeID,
		bus:         bus,
		psrpcClient: psrpcClient,
		store:       store,
		roomService: rs,
	}
}

func (s *SIPService) CreateSIPTrunk(ctx context.Context, req *livekit.CreateSIPTrunkRequest) (*livekit.SIPTrunkInfo, error) {
	if err := EnsureSIPAdminPermission(ctx); err != nil {
		return nil, twirpAuthError(err)
	}
	if s.store == nil {
		return nil, ErrSIPNotConnected
	}
	if len(req.InboundNumbersRegex) != 0 {
		return nil, fmt.Errorf("Trunks with InboundNumbersRegex are deprecated. Use InboundNumbers instead.")
	}

	// Keep ID empty, so that validation can print "<new>" instead of a non-existent ID in the error.
	info := &livekit.SIPTrunkInfo{
		InboundAddresses: req.InboundAddresses,
		OutboundAddress:  req.OutboundAddress,
		OutboundNumber:   req.OutboundNumber,
		InboundNumbers:   req.InboundNumbers,
		InboundUsername:  req.InboundUsername,
		InboundPassword:  req.InboundPassword,
		OutboundUsername: req.OutboundUsername,
		OutboundPassword: req.OutboundPassword,
		Name:             req.Name,
		Metadata:         req.Metadata,
	}
	if err := info.Validate(); err != nil {
		return nil, err
	}

	// Validate all trunks including the new one first.
	list, err := s.store.ListSIPInboundTrunk(ctx)
	if err != nil {
		return nil, err
	}
	list = append(list, info.AsInbound())
	if err = sip.ValidateTrunks(list); err != nil {
		return nil, err
	}

	// Now we can generate ID and store.
	info.SipTrunkId = guid.New(utils.SIPTrunkPrefix)
	if err := s.store.StoreSIPTrunk(ctx, info); err != nil {
		return nil, err
	}
	return info, nil
}

func (s *SIPService) CreateSIPInboundTrunk(ctx context.Context, req *livekit.CreateSIPInboundTrunkRequest) (*livekit.SIPInboundTrunkInfo, error) {
	if err := EnsureSIPAdminPermission(ctx); err != nil {
		return nil, twirpAuthError(err)
	}
	if s.store == nil {
		return nil, ErrSIPNotConnected
	}
	if err := req.Validate(); err != nil {
		return nil, err
	}
	info := req.Trunk
	AppendLogFields(ctx, "trunk", logger.Proto(info))

	// Keep ID empty still, so that validation can print "<new>" instead of a non-existent ID in the error.

	// Validate all trunks including the new one first.
	list, err := s.store.ListSIPInboundTrunk(ctx)
	if err != nil {
		return nil, err
	}
	list = append(list, info)
	if err = sip.ValidateTrunks(list); err != nil {
		return nil, err
	}

	// Now we can generate ID and store.
	info.SipTrunkId = guid.New(utils.SIPTrunkPrefix)
	if err := s.store.StoreSIPInboundTrunk(ctx, info); err != nil {
		return nil, err
	}
	return info, nil
}

func (s *SIPService) CreateSIPOutboundTrunk(ctx context.Context, req *livekit.CreateSIPOutboundTrunkRequest) (*livekit.SIPOutboundTrunkInfo, error) {
	if err := EnsureSIPAdminPermission(ctx); err != nil {
		return nil, twirpAuthError(err)
	}
	if s.store == nil {
		return nil, ErrSIPNotConnected
	}
	if err := req.Validate(); err != nil {
		return nil, err
	}
	info := req.Trunk
	AppendLogFields(ctx, "trunk", logger.Proto(info))

	// No additional validation needed for outbound.
	info.SipTrunkId = guid.New(utils.SIPTrunkPrefix)
	if err := s.store.StoreSIPOutboundTrunk(ctx, info); err != nil {
		return nil, err
	}
	return info, nil
}

func (s *SIPService) UpdateSIPInboundTrunk(ctx context.Context, req *livekit.UpdateSIPInboundTrunkRequest) (*livekit.SIPInboundTrunkInfo, error) {
	if err := EnsureSIPAdminPermission(ctx); err != nil {
		return nil, twirpAuthError(err)
	}
	if s.store == nil {
		return nil, ErrSIPNotConnected
	}
	if err := req.Validate(); err != nil {
		return nil, err
	}

	AppendLogFields(ctx,
		"request", logger.Proto(req),
		"trunkID", req.SipTrunkId,
	)

	// Validate all trunks including the new one first.
	list, err := s.store.ListSIPInboundTrunk(ctx)
	if err != nil {
		return nil, err
	}
	i := slices.IndexFunc(list, func(info2 *livekit.SIPInboundTrunkInfo) bool {
		return req.SipTrunkId == info2.SipTrunkId
	})
	if i < 0 {
		return nil, ErrSIPTrunkNotFound
	}
	info := list[i]
	switch a := req.Action.(type) {
	default:
		return nil, errors.New("missing or unsupported action")
	case livekit.UpdateSIPInboundTrunkRequestAction:
		if err = a.Apply(info); err != nil {
			return nil, err
		}
	}
	list[i] = info
	if err = sip.ValidateTrunks(list); err != nil {
		return nil, err
	}
	if err := s.store.StoreSIPInboundTrunk(ctx, info); err != nil {
		return nil, err
	}
	return info, nil
}

func (s *SIPService) UpdateSIPOutboundTrunk(ctx context.Context, req *livekit.UpdateSIPOutboundTrunkRequest) (*livekit.SIPOutboundTrunkInfo, error) {
	if err := EnsureSIPAdminPermission(ctx); err != nil {
		return nil, twirpAuthError(err)
	}
	if s.store == nil {
		return nil, ErrSIPNotConnected
	}
	if err := req.Validate(); err != nil {
		return nil, err
	}

	AppendLogFields(ctx,
		"request", logger.Proto(req),
		"trunkID", req.SipTrunkId,
	)

	info, err := s.store.LoadSIPOutboundTrunk(ctx, req.SipTrunkId)
	if err != nil {
		return nil, err
	}
	switch a := req.Action.(type) {
	default:
		return nil, errors.New("missing or unsupported action")
	case livekit.UpdateSIPOutboundTrunkRequestAction:
		if err = a.Apply(info); err != nil {
			return nil, err
		}
	}
	// No additional validation needed for outbound.
	if err := s.store.StoreSIPOutboundTrunk(ctx, info); err != nil {
		return nil, err
	}
	return info, nil
}

func (s *SIPService) GetSIPInboundTrunk(ctx context.Context, req *livekit.GetSIPInboundTrunkRequest) (*livekit.GetSIPInboundTrunkResponse, error) {
	if err := EnsureSIPAdminPermission(ctx); err != nil {
		return nil, twirpAuthError(err)
	}
	if s.store == nil {
		return nil, ErrSIPNotConnected
	}

	trunk, err := s.store.LoadSIPInboundTrunk(ctx, req.SipTrunkId)
	if err != nil {
		return nil, err
	}

	return &livekit.GetSIPInboundTrunkResponse{Trunk: trunk}, nil
}

func (s *SIPService) GetSIPOutboundTrunk(ctx context.Context, req *livekit.GetSIPOutboundTrunkRequest) (*livekit.GetSIPOutboundTrunkResponse, error) {
	if err := EnsureSIPAdminPermission(ctx); err != nil {
		return nil, twirpAuthError(err)
	}
	if s.store == nil {
		return nil, ErrSIPNotConnected
	}
	AppendLogFields(ctx, "trunkID", req.SipTrunkId)

	trunk, err := s.store.LoadSIPOutboundTrunk(ctx, req.SipTrunkId)
	if err != nil {
		return nil, err
	}

	return &livekit.GetSIPOutboundTrunkResponse{Trunk: trunk}, nil
}

// deprecated: ListSIPTrunk will be removed in the future
func (s *SIPService) ListSIPTrunk(ctx context.Context, req *livekit.ListSIPTrunkRequest) (*livekit.ListSIPTrunkResponse, error) {
	if err := EnsureSIPAdminPermission(ctx); err != nil {
		return nil, twirpAuthError(err)
	}
	if s.store == nil {
		return nil, ErrSIPNotConnected
	}

	trunks, err := s.store.ListSIPTrunk(ctx)
	if err != nil {
		return nil, err
	}

	return &livekit.ListSIPTrunkResponse{Items: trunks}, nil
}

func (s *SIPService) ListSIPInboundTrunk(ctx context.Context, req *livekit.ListSIPInboundTrunkRequest) (*livekit.ListSIPInboundTrunkResponse, error) {
	if err := EnsureSIPAdminPermission(ctx); err != nil {
		return nil, twirpAuthError(err)
	}
	if s.store == nil {
		return nil, ErrSIPNotConnected
	}

	trunks, err := s.store.ListSIPInboundTrunk(ctx)
	if err != nil {
		return nil, err
	}

	return &livekit.ListSIPInboundTrunkResponse{Items: trunks}, nil
}

func (s *SIPService) ListSIPOutboundTrunk(ctx context.Context, req *livekit.ListSIPOutboundTrunkRequest) (*livekit.ListSIPOutboundTrunkResponse, error) {
	if err := EnsureSIPAdminPermission(ctx); err != nil {
		return nil, twirpAuthError(err)
	}
	if s.store == nil {
		return nil, ErrSIPNotConnected
	}

	trunks, err := s.store.ListSIPOutboundTrunk(ctx)
	if err != nil {
		return nil, err
	}

	return &livekit.ListSIPOutboundTrunkResponse{Items: trunks}, nil
}

func (s *SIPService) DeleteSIPTrunk(ctx context.Context, req *livekit.DeleteSIPTrunkRequest) (*livekit.SIPTrunkInfo, error) {
	if err := EnsureSIPAdminPermission(ctx); err != nil {
		return nil, twirpAuthError(err)
	}
	if s.store == nil {
		return nil, ErrSIPNotConnected
	}

	AppendLogFields(ctx, "trunkID", req.SipTrunkId)
	if err := s.store.DeleteSIPTrunk(ctx, req.SipTrunkId); err != nil {
		return nil, err
	}

	return &livekit.SIPTrunkInfo{SipTrunkId: req.SipTrunkId}, nil
}

func (s *SIPService) CreateSIPDispatchRule(ctx context.Context, req *livekit.CreateSIPDispatchRuleRequest) (*livekit.SIPDispatchRuleInfo, error) {
	if err := EnsureSIPAdminPermission(ctx); err != nil {
		return nil, twirpAuthError(err)
	}
	if s.store == nil {
		return nil, ErrSIPNotConnected
	}
	if err := req.Validate(); err != nil {
		return nil, err
	}

	AppendLogFields(ctx,
		"request", logger.Proto(req),
		"trunkID", req.TrunkIds,
	)
	// Keep ID empty, so that validation can print "<new>" instead of a non-existent ID in the error.
	info := req.DispatchRuleInfo()

	// Validate all rules including the new one first.
	list, err := s.store.ListSIPDispatchRule(ctx)
	if err != nil {
		return nil, err
	}
	list = append(list, info)
	if err = sip.ValidateDispatchRules(list); err != nil {
		return nil, err
	}

	// Now we can generate ID and store.
	info.SipDispatchRuleId = guid.New(utils.SIPDispatchRulePrefix)
	if err := s.store.StoreSIPDispatchRule(ctx, info); err != nil {
		return nil, err
	}
	return info, nil
}

func (s *SIPService) UpdateSIPDispatchRule(ctx context.Context, req *livekit.UpdateSIPDispatchRuleRequest) (*livekit.SIPDispatchRuleInfo, error) {
	if err := EnsureSIPAdminPermission(ctx); err != nil {
		return nil, twirpAuthError(err)
	}
	if s.store == nil {
		return nil, ErrSIPNotConnected
	}
	if err := req.Validate(); err != nil {
		return nil, err
	}

	AppendLogFields(ctx,
		"request", logger.Proto(req),
		"ruleID", req.SipDispatchRuleId,
	)

	// Validate all rules including the new one first.
	list, err := s.store.ListSIPDispatchRule(ctx)
	if err != nil {
		return nil, err
	}
	i := slices.IndexFunc(list, func(info2 *livekit.SIPDispatchRuleInfo) bool {
		return req.SipDispatchRuleId == info2.SipDispatchRuleId
	})
	if i < 0 {
		return nil, ErrSIPDispatchRuleNotFound
	}
	info := list[i]
	switch a := req.Action.(type) {
	default:
		return nil, errors.New("missing or unsupported action")
	case livekit.UpdateSIPDispatchRuleRequestAction:
		if err = a.Apply(info); err != nil {
			return nil, err
		}
	}
	list[i] = info
	if err = sip.ValidateDispatchRules(list); err != nil {
		return nil, err
	}
	if err := s.store.StoreSIPDispatchRule(ctx, info); err != nil {
		return nil, err
	}
	return info, nil
}

func (s *SIPService) ListSIPDispatchRule(ctx context.Context, req *livekit.ListSIPDispatchRuleRequest) (*livekit.ListSIPDispatchRuleResponse, error) {
	if err := EnsureSIPAdminPermission(ctx); err != nil {
		return nil, twirpAuthError(err)
	}
	if s.store == nil {
		return nil, ErrSIPNotConnected
	}

	rules, err := s.store.ListSIPDispatchRule(ctx)
	if err != nil {
		return nil, err
	}

	return &livekit.ListSIPDispatchRuleResponse{Items: rules}, nil
}

func (s *SIPService) DeleteSIPDispatchRule(ctx context.Context, req *livekit.DeleteSIPDispatchRuleRequest) (*livekit.SIPDispatchRuleInfo, error) {
	if err := EnsureSIPAdminPermission(ctx); err != nil {
		return nil, twirpAuthError(err)
	}
	if s.store == nil {
		return nil, ErrSIPNotConnected
	}

	info, err := s.store.LoadSIPDispatchRule(ctx, req.SipDispatchRuleId)
	if err != nil {
		return nil, err
	}

	if err = s.store.DeleteSIPDispatchRule(ctx, info); err != nil {
		return nil, err
	}

	return info, nil
}

func (s *SIPService) CreateSIPParticipant(ctx context.Context, req *livekit.CreateSIPParticipantRequest) (*livekit.SIPParticipantInfo, error) {
	unlikelyLogger := logger.GetLogger().WithUnlikelyValues("room", req.RoomName, "sipTrunk", req.SipTrunkId, "toUser", req.SipCallTo)
	ireq, err := s.CreateSIPParticipantRequest(ctx, req, "", "", "", "")
	if err != nil {
		unlikelyLogger.Errorw("cannot create sip participant request", err)
		return nil, err
	}
	unlikelyLogger = unlikelyLogger.WithValues(
		"callID", ireq.SipCallId,
		"fromUser", ireq.Number,
		"toHost", ireq.Address,
	)
	AppendLogFields(ctx,
		"room", req.RoomName,
		"toUser", req.SipCallTo,
		"trunkID", req.SipTrunkId,
		"callID", ireq.SipCallId,
		"fromUser", ireq.Number,
		"toHost", ireq.Address,
	)

	// CreateSIPParticipant will wait for LiveKit Participant to be created and that can take some time.
	// Thus, we must set a higher deadline for it, if it's not set already.
	// TODO: support context timeouts in psrpc
	timeout := 30 * time.Second
	if deadline, ok := ctx.Deadline(); ok {
		timeout = time.Until(deadline)
	} else {
		var cancel func()
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}
	resp, err := s.psrpcClient.CreateSIPParticipant(ctx, "", ireq, psrpc.WithRequestTimeout(timeout))
	if err != nil {
		unlikelyLogger.Errorw("cannot update sip participant", err)
		return nil, err
	}
	return &livekit.SIPParticipantInfo{
		ParticipantId:       resp.ParticipantId,
		ParticipantIdentity: resp.ParticipantIdentity,
		RoomName:            req.RoomName,
		SipCallId:           ireq.SipCallId,
	}, nil
}

func (s *SIPService) CreateSIPParticipantRequest(ctx context.Context, req *livekit.CreateSIPParticipantRequest, projectID, host, wsUrl, token string) (*rpc.InternalCreateSIPParticipantRequest, error) {
	if err := EnsureSIPCallPermission(ctx); err != nil {
		return nil, twirpAuthError(err)
	}
	if s.store == nil {
		return nil, ErrSIPNotConnected
	}
	if err := req.Validate(); err != nil {
		return nil, err
	}
	callID := sip.NewCallID()
	log := logger.GetLogger().WithUnlikelyValues(
		"callID", callID,
		"room", req.RoomName,
		"sipTrunk", req.SipTrunkId,
		"toUser", req.SipCallTo,
	)
	if projectID != "" {
		log = log.WithValues("projectID", projectID)
	}

	trunk, err := s.store.LoadSIPOutboundTrunk(ctx, req.SipTrunkId)
	if err != nil {
		log.Errorw("cannot get trunk to update sip participant", err)
		return nil, err
	}
	return rpc.NewCreateSIPParticipantRequest(projectID, callID, host, wsUrl, token, req, trunk)
}

func (s *SIPService) TransferSIPParticipant(ctx context.Context, req *livekit.TransferSIPParticipantRequest) (*emptypb.Empty, error) {
	log := logger.GetLogger().WithUnlikelyValues("room", req.RoomName, "participant", req.ParticipantIdentity)
	ireq, err := s.transferSIPParticipantRequest(ctx, req)
	if err != nil {
		log.Errorw("cannot create transfer sip participant request", err)
		return nil, err
	}
	AppendLogFields(ctx,
		"room", req.RoomName,
		"participant", req.ParticipantIdentity,
		"transferTo", req.TransferTo,
		"playDialtone", req.PlayDialtone,
	)

	timeout := 30 * time.Second
	if deadline, ok := ctx.Deadline(); ok {
		timeout = time.Until(deadline)
	} else {
		var cancel func()
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}
	_, err = s.psrpcClient.TransferSIPParticipant(ctx, ireq.SipCallId, ireq, psrpc.WithRequestTimeout(timeout))
	if err != nil {
		log.Errorw("cannot transfer sip participant", err)
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

func (s *SIPService) transferSIPParticipantRequest(ctx context.Context, req *livekit.TransferSIPParticipantRequest) (*rpc.InternalTransferSIPParticipantRequest, error) {
	if req.RoomName == "" {
		return nil, psrpc.NewErrorf(psrpc.InvalidArgument, "Missing room name")
	}

	if req.ParticipantIdentity == "" {
		return nil, psrpc.NewErrorf(psrpc.InvalidArgument, "Missing participant identity")
	}

	if err := EnsureSIPCallPermission(ctx); err != nil {
		return nil, twirpAuthError(err)
	}
	if err := EnsureAdminPermission(ctx, livekit.RoomName(req.RoomName)); err != nil {
		return nil, twirpAuthError(err)
	}
	if err := req.Validate(); err != nil {
		return nil, err
	}

	resp, err := s.roomService.GetParticipant(ctx, &livekit.RoomParticipantIdentity{
		Room:     req.RoomName,
		Identity: req.ParticipantIdentity,
	})

	if err != nil {
		return nil, err
	}

	callID, ok := resp.Attributes[livekit.AttrSIPCallID]
	if !ok {
		return nil, psrpc.NewErrorf(psrpc.InvalidArgument, "no SIP session associated with participant")
	}

	return &rpc.InternalTransferSIPParticipantRequest{
		SipCallId:    callID,
		TransferTo:   req.TransferTo,
		PlayDialtone: req.PlayDialtone,
	}, nil
}
