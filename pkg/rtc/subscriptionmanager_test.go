/*
 * Copyright 2023 LiveKit, Inc
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package rtc

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/livekit/livekit-server/pkg/rtc/types"
	"github.com/livekit/livekit-server/pkg/rtc/types/typesfakes"
	"github.com/livekit/livekit-server/pkg/telemetry/telemetryfakes"
	"github.com/livekit/livekit-server/pkg/utils"
	"github.com/livekit/protocol/livekit"
	"github.com/livekit/protocol/logger"
)

func init() {
	reconcileInterval = 50 * time.Millisecond
	notFoundTimeout = 200 * time.Millisecond
	subscriptionTimeout = 200 * time.Millisecond
}

const (
	subSettleTimeout = 300 * time.Millisecond
	subCheckInterval = 10 * time.Millisecond
)

func TestSubscribe(t *testing.T) {
	t.Run("happy path subscribe", func(t *testing.T) {
		sm := newTestSubscriptionManager(t)
		defer sm.Close(false)
		resolver := newTestResolver(true, nil)
		sm.params.TrackResolver = resolver.Resolve
		subCount := atomic.Int32{}
		failed := atomic.Bool{}
		sm.params.OnTrackSubscribed = func(subTrack types.SubscribedTrack) {
			subCount.Add(1)
		}
		sm.params.OnSubcriptionError = func(trackID livekit.TrackID) {
			failed.Store(true)
		}
		numParticipantSubscribed := atomic.Int32{}
		numParticipantUnsubscribed := atomic.Int32{}
		sm.OnSubscribeStatusChanged(func(pubID livekit.ParticipantID, subscribed bool) {
			if subscribed {
				numParticipantSubscribed.Add(1)
			} else {
				numParticipantUnsubscribed.Add(1)
			}
		})

		sm.SubscribeToTrack("track", "pub", "pubID")
		s := sm.subscriptions["track"]
		require.True(t, s.isDesired())
		require.Eventually(t, func() bool {
			return subCount.Load() == 1
		}, subSettleTimeout, subCheckInterval, "track was not subscribed")

		require.NotNil(t, s.getSubscribedTrack())
		require.Len(t, sm.GetSubscribedTracks(), 1)
		require.Len(t, sm.GetSubscribedParticipants(), 1)
		require.Equal(t, "pubID", string(sm.GetSubscribedParticipants()[0]))

		// ensure telemetry events are sent
		tm := sm.params.Telemetry.(*telemetryfakes.FakeTelemetryService)
		require.Equal(t, 1, tm.TrackSubscribeRequestedCallCount())

		// ensure bound
		setTestSubscribedTrackBound(t, s.getSubscribedTrack())

		require.Eventually(t, func() bool {
			return !s.needsBind()
		}, subSettleTimeout, subCheckInterval, "track was not bound")

		// telemetry event should have been sent
		require.Equal(t, 1, tm.TrackSubscribedCallCount())

		time.Sleep(notFoundTimeout)
		require.False(t, failed.Load())

		// ensure its resilience after being closed
		setTestSubscribedTrackClosed(t, s.getSubscribedTrack(), false)
		require.True(t, s.needsSubscribe())

		require.Eventually(t, func() bool {
			return s.isDesired() && !s.needsSubscribe()
		}, subSettleTimeout, subCheckInterval, "track was not resubscribed")

		// was subscribed twice, unsubscribed once (due to close)
		require.Equal(t, int32(2), numParticipantSubscribed.Load())
		require.Equal(t, int32(1), numParticipantUnsubscribed.Load())
	})

	t.Run("no track permission", func(t *testing.T) {
		sm := newTestSubscriptionManager(t)
		defer sm.Close(false)
		resolver := newTestResolver(false, nil)
		sm.params.TrackResolver = resolver.Resolve
		failed := atomic.Bool{}
		sm.params.OnSubcriptionError = func(trackID livekit.TrackID) {
			failed.Store(true)
		}

		sm.SubscribeToTrack("track", "pub", "pubID")
		s := sm.subscriptions["track"]
		require.Eventually(t, func() bool {
			return !s.getHasPermission()
		}, subSettleTimeout, subCheckInterval, "should not have permission to subscribe")

		time.Sleep(subscriptionTimeout)

		// should not have called failed callbacks, isDesired remains unchanged
		require.True(t, s.isDesired())
		require.False(t, failed.Load())
		require.True(t, s.needsSubscribe())
		require.Len(t, sm.GetSubscribedTracks(), 0)

		// trackSubscribed telemetry not sent
		tm := sm.params.Telemetry.(*telemetryfakes.FakeTelemetryService)
		require.Equal(t, 1, tm.TrackSubscribeRequestedCallCount())
		require.Equal(t, 0, tm.TrackSubscribedCallCount())

		// give permissions now
		resolver.lock.Lock()
		resolver.hasPermission = true
		resolver.lock.Unlock()

		require.Eventually(t, func() bool {
			return !s.needsSubscribe()
		}, subSettleTimeout, subCheckInterval, "should be subscribed")

		require.Len(t, sm.GetSubscribedTracks(), 1)
	})

	t.Run("publisher left", func(t *testing.T) {
		sm := newTestSubscriptionManager(t)
		defer sm.Close(false)
		resolver := newTestResolver(true, nil)
		sm.params.TrackResolver = resolver.Resolve
		failed := atomic.Bool{}
		sm.params.OnSubcriptionError = func(trackID livekit.TrackID) {
			failed.Store(true)
		}

		sm.SubscribeToTrack("track", "pub", "pubID")
		s := sm.subscriptions["track"]
		require.Eventually(t, func() bool {
			return !s.needsSubscribe()
		}, subSettleTimeout, subCheckInterval, "should be subscribed")

		resolver.lock.Lock()
		resolver.err = ErrTrackNotFound
		resolver.lock.Unlock()

		// publisher triggers close
		setTestSubscribedTrackClosed(t, s.getSubscribedTrack(), false)

		require.Eventually(t, func() bool {
			return !s.isDesired()
		}, subSettleTimeout, subCheckInterval, "isDesired not set to false")
	})
}

func TestUnsubscribe(t *testing.T) {
	sm := newTestSubscriptionManager(t)
	defer sm.Close(false)
	unsubCount := atomic.Int32{}
	sm.params.OnTrackUnsubscribed = func(subTrack types.SubscribedTrack) {
		unsubCount.Add(1)
	}

	resolver := newTestResolver(true, nil)

	s := &trackSubscription{
		trackID:           "track",
		desired:           true,
		subscriberID:      sm.params.Participant.ID(),
		publisherID:       "pubID",
		publisherIdentity: "pub",
		hasPermission:     true,
		bound:             true,
		logger:            logger.GetLogger(),
	}
	// a bunch of unfortunate manual wiring
	res, err := resolver.Resolve("sub", s.publisherID, s.trackID)
	require.NoError(t, err)
	res.TrackChangeNotifier.AddObserver(string(sm.params.Participant.ID()), func() {})
	s.changeNotifier = res.TrackChangeNotifier
	st, err := res.Track.AddSubscriber(sm.params.Participant)
	require.NoError(t, err)
	s.subscribedTrack = st
	st.OnClose(func(willBeResumed bool) {
		sm.handleSubscribedTrackClose(s, willBeResumed)
	})
	res.Track.(*typesfakes.FakeMediaTrack).RemoveSubscriberStub = func(pID livekit.ParticipantID, willBeResumed bool) {
		setTestSubscribedTrackClosed(t, st, willBeResumed)
	}

	sm.lock.Lock()
	sm.subscriptions["track"] = s
	sm.lock.Unlock()

	require.False(t, s.needsSubscribe())
	require.False(t, s.needsUnsubscribe())

	// unsubscribe
	sm.UnsubscribeFromTrack("track")
	require.False(t, s.isDesired())

	require.Eventually(t, func() bool {
		return !s.needsUnsubscribe()
	}, subSettleTimeout, subCheckInterval, "track was not unsubscribed")

	// no traces should be left
	require.Len(t, sm.GetSubscribedTracks(), 0)
	sm.lock.RLock()
	require.Len(t, sm.subscriptions, 0)
	sm.lock.RUnlock()
	require.False(t, res.TrackChangeNotifier.HasObservers())

	tm := sm.params.Telemetry.(*telemetryfakes.FakeTelemetryService)
	require.Equal(t, 1, tm.TrackUnsubscribedCallCount())
}

func TestSubscribeStatusChanged(t *testing.T) {
	sm := newTestSubscriptionManager(t)
	defer sm.Close(false)
	resolver := newTestResolver(true, nil)
	sm.params.TrackResolver = resolver.Resolve
	numParticipantSubscribed := atomic.Int32{}
	numParticipantUnsubscribed := atomic.Int32{}
	sm.OnSubscribeStatusChanged(func(pubID livekit.ParticipantID, subscribed bool) {
		if subscribed {
			numParticipantSubscribed.Add(1)
		} else {
			numParticipantUnsubscribed.Add(1)
		}
	})

	sm.SubscribeToTrack("track1", "pub", "pubID")
	sm.SubscribeToTrack("track2", "pub", "pubID")
	s1 := sm.subscriptions["track1"]
	s2 := sm.subscriptions["track2"]
	require.Eventually(t, func() bool {
		return !s1.needsSubscribe() && !s2.needsSubscribe()
	}, subSettleTimeout, subCheckInterval, "track1 and track2 should be subscribed")
	st1 := s1.getSubscribedTrack()
	st1.OnClose(func(willBeResumed bool) {
		sm.handleSubscribedTrackClose(s1, willBeResumed)
	})
	st2 := s2.getSubscribedTrack()
	st2.OnClose(func(willBeResumed bool) {
		sm.handleSubscribedTrackClose(s2, willBeResumed)
	})
	st1.MediaTrack().(*typesfakes.FakeMediaTrack).RemoveSubscriberStub = func(pID livekit.ParticipantID, willBeResumed bool) {
		setTestSubscribedTrackClosed(t, st1, willBeResumed)
	}
	st2.MediaTrack().(*typesfakes.FakeMediaTrack).RemoveSubscriberStub = func(pID livekit.ParticipantID, willBeResumed bool) {
		setTestSubscribedTrackClosed(t, st2, willBeResumed)
	}

	require.Equal(t, int32(1), numParticipantSubscribed.Load())
	require.Equal(t, int32(0), numParticipantUnsubscribed.Load())

	// now unsubscribe track2, no event should be fired
	sm.UnsubscribeFromTrack("track2")
	require.Eventually(t, func() bool {
		return !s2.needsUnsubscribe()
	}, subSettleTimeout, subCheckInterval, "track2 should be unsubscribed")
	require.Equal(t, int32(0), numParticipantUnsubscribed.Load())

	// unsubscribe track1, expect event
	sm.UnsubscribeFromTrack("track1")
	require.Eventually(t, func() bool {
		return !s1.needsUnsubscribe()
	}, subSettleTimeout, subCheckInterval, "track1 should be unsubscribed")
	require.Equal(t, int32(1), numParticipantUnsubscribed.Load())
}

// clients may send update subscribed settings prior to subscription events coming through
// settings should be persisted and used when the subscription does take place.
func TestUpdateSettingsBeforeSubscription(t *testing.T) {
	sm := newTestSubscriptionManager(t)
	defer sm.Close(false)
	resolver := newTestResolver(true, nil)
	sm.params.TrackResolver = resolver.Resolve

	settings := &livekit.UpdateTrackSettings{
		Disabled: true,
		Width:    100,
		Height:   100,
	}
	sm.UpdateSubscribedTrackSettings("track", settings)

	sm.SubscribeToTrack("track", "pub", "pubID")

	s := sm.subscriptions["track"]
	require.Eventually(t, func() bool {
		return !s.needsSubscribe()
	}, subSettleTimeout, subCheckInterval, "track should be subscribed")

	st := s.getSubscribedTrack().(*typesfakes.FakeSubscribedTrack)
	require.Equal(t, 1, st.UpdateSubscriberSettingsCallCount())
	applied := st.UpdateSubscriberSettingsArgsForCall(0)
	require.Equal(t, settings.Disabled, applied.Disabled)
	require.Equal(t, settings.Width, applied.Width)
	require.Equal(t, settings.Height, applied.Height)
}

func newTestSubscriptionManager(t *testing.T) *SubscriptionManager {
	p := &typesfakes.FakeLocalParticipant{}
	p.CanSubscribeReturns(true)
	p.IDReturns("subID")
	p.IdentityReturns("sub")
	return NewSubscriptionManager(SubscriptionManagerParams{
		Participant:         p,
		Logger:              logger.GetLogger(),
		OnTrackSubscribed:   func(subTrack types.SubscribedTrack) {},
		OnTrackUnsubscribed: func(subTrack types.SubscribedTrack) {},
		OnSubcriptionError:  func(trackID livekit.TrackID) {},
		TrackResolver: func(identity livekit.ParticipantIdentity, pID livekit.ParticipantID, trackID livekit.TrackID) (types.MediaResolverResult, error) {
			return types.MediaResolverResult{}, ErrTrackNotFound
		},
		Telemetry: &telemetryfakes.FakeTelemetryService{},
	})
}

type testResolver struct {
	lock          sync.Mutex
	hasPermission bool
	err           error
}

func newTestResolver(hasPermission bool, err error) *testResolver {
	return &testResolver{
		hasPermission: hasPermission,
		err:           err,
	}
}

func (t *testResolver) Resolve(identity livekit.ParticipantIdentity, pID livekit.ParticipantID, trackID livekit.TrackID) (types.MediaResolverResult, error) {
	t.lock.Lock()
	defer t.lock.Unlock()
	if t.err != nil {
		return types.MediaResolverResult{}, t.err
	}
	mt := &typesfakes.FakeMediaTrack{}
	st := &typesfakes.FakeSubscribedTrack{}
	st.IDReturns(trackID)
	st.PublisherIDReturns(pID)
	st.PublisherIdentityReturns(identity)
	mt.AddSubscriberReturns(st, nil)
	st.MediaTrackReturns(mt)
	return types.MediaResolverResult{
		Track:               mt,
		TrackChangeNotifier: utils.NewChangeNotifier(),
		HasPermission:       t.hasPermission,
	}, nil
}

func setTestSubscribedTrackBound(t *testing.T, st types.SubscribedTrack) {
	fst, ok := st.(*typesfakes.FakeSubscribedTrack)
	require.True(t, ok)

	for i := 0; i < fst.AddOnBindCallCount(); i++ {
		fst.AddOnBindArgsForCall(i)()
	}
}

func setTestSubscribedTrackClosed(t *testing.T, st types.SubscribedTrack, willBeResumed bool) {
	fst, ok := st.(*typesfakes.FakeSubscribedTrack)
	require.True(t, ok)

	fst.OnCloseArgsForCall(0)(willBeResumed)
}