module github.com/livekit/livekit-server

go 1.22.7

require (
	github.com/bep/debounce v1.2.1
	github.com/d5/tengo/v2 v2.17.0
	github.com/dustin/go-humanize v1.0.1
	github.com/elliotchance/orderedmap/v2 v2.4.0
	github.com/florianl/go-tc v0.4.4
	github.com/frostbyte73/core v0.0.13
	github.com/gammazero/deque v0.2.1
	github.com/gammazero/workerpool v1.1.3
	github.com/google/wire v0.6.0
	github.com/gorilla/websocket v1.5.3
	github.com/hashicorp/go-version v1.7.0
	github.com/hashicorp/golang-lru/v2 v2.0.7
	github.com/jellydator/ttlcache/v3 v3.3.0
	github.com/jxskiss/base62 v1.1.0
	github.com/livekit/mageutil v0.0.0-20230125210925-54e8a70427c1
	github.com/livekit/mediatransportutil v0.0.0-20240730083616-559fa5ece598
	github.com/livekit/protocol v1.27.2-0.20241107143334-083fcac1318d
	github.com/livekit/psrpc v0.6.1-0.20240924010758-9f0a4268a3b9
	github.com/mackerelio/go-osstat v0.2.5
	github.com/magefile/mage v1.15.0
	github.com/maxbrunsfeld/counterfeiter/v6 v6.9.0
	github.com/mitchellh/go-homedir v1.1.0
	github.com/olekukonko/tablewriter v0.0.5
	github.com/ory/dockertest/v3 v3.11.0
	github.com/pion/dtls/v2 v2.2.7
	github.com/pion/dtls/v3 v3.0.4
	github.com/pion/ice/v2 v2.3.13
	github.com/pion/ice/v4 v4.0.2
	github.com/pion/interceptor v0.1.37
	github.com/pion/rtcp v1.2.14
	github.com/pion/rtp v1.8.9
	github.com/pion/rtp/v2 v2.0.0
	github.com/pion/sctp v1.8.33
	github.com/pion/sdp/v3 v3.0.9
	github.com/pion/transport/v2 v2.2.3
	github.com/pion/transport/v3 v3.0.7
	github.com/pion/turn/v2 v2.1.3
	github.com/pion/turn/v4 v4.0.0
	github.com/pion/webrtc/v3 v3.2.28
	github.com/pion/webrtc/v4 v4.0.2
	github.com/pkg/errors v0.9.1
	github.com/prometheus/client_golang v1.20.4
	github.com/redis/go-redis/v9 v9.6.2
	github.com/rs/cors v1.11.1
	github.com/stretchr/testify v1.9.0
	github.com/thoas/go-funk v0.9.3
	github.com/twitchtv/twirp v8.1.3+incompatible
	github.com/ua-parser/uap-go v0.0.0-20240611065828-3a4781585db6
	github.com/urfave/cli/v2 v2.27.4
	github.com/urfave/negroni/v3 v3.1.1
	go.uber.org/atomic v1.11.0
	go.uber.org/multierr v1.11.0
	go.uber.org/zap v1.27.0
	golang.org/x/exp v0.0.0-20241009180824-f66d83c29e7c
	golang.org/x/sync v0.8.0
	google.golang.org/protobuf v1.35.1
	gopkg.in/yaml.v3 v3.0.1
)

// replace github.com/livekit/protocol => ../protocol

require (
	buf.build/gen/go/bufbuild/protovalidate/protocolbuffers/go v1.33.0-20240401165935-b983156c5e99.1 // indirect
	dario.cat/mergo v1.0.0 // indirect
	github.com/Azure/go-ansiterm v0.0.0-20230124172434-306776ec8161 // indirect
	github.com/Microsoft/go-winio v0.6.2 // indirect
	github.com/Nvveen/Gotty v0.0.0-20120604004816-cd527374f1e5 // indirect
	github.com/antlr4-go/antlr/v4 v4.13.0 // indirect
	github.com/benbjohnson/clock v1.3.5 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/bufbuild/protovalidate-go v0.6.1 // indirect
	github.com/bufbuild/protoyaml-go v0.1.9 // indirect
	github.com/cenkalti/backoff/v4 v4.3.0 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/containerd/continuity v0.4.3 // indirect
	github.com/cpuguy83/go-md2man/v2 v2.0.4 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
	github.com/docker/cli v26.1.4+incompatible // indirect
	github.com/docker/docker v27.1.1+incompatible // indirect
	github.com/docker/go-connections v0.5.0 // indirect
	github.com/docker/go-units v0.5.0 // indirect
	github.com/fsnotify/fsnotify v1.7.0 // indirect
	github.com/go-jose/go-jose/v3 v3.0.3 // indirect
	github.com/go-logr/logr v1.4.2 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/google/cel-go v0.20.1 // indirect
	github.com/google/go-cmp v0.6.0 // indirect
	github.com/google/shlex v0.0.0-20191202100458-e7afc7fbc510 // indirect
	github.com/google/subcommands v1.2.0 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/hashicorp/go-cleanhttp v0.5.2 // indirect
	github.com/hashicorp/go-retryablehttp v0.7.7 // indirect
	github.com/hashicorp/golang-lru v0.5.4 // indirect
	github.com/josharian/native v1.1.0 // indirect
	github.com/klauspost/compress v1.17.11 // indirect
	github.com/klauspost/cpuid/v2 v2.2.6 // indirect
	github.com/lithammer/shortuuid/v4 v4.0.0 // indirect
	github.com/mattn/go-runewidth v0.0.9 // indirect
	github.com/mdlayher/netlink v1.7.1 // indirect
	github.com/mdlayher/socket v0.4.0 // indirect
	github.com/mitchellh/mapstructure v1.5.0 // indirect
	github.com/moby/docker-image-spec v1.3.1 // indirect
	github.com/moby/term v0.5.0 // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/nats-io/nats.go v1.37.0 // indirect
	github.com/nats-io/nkeys v0.4.7 // indirect
	github.com/nats-io/nuid v1.0.1 // indirect
	github.com/opencontainers/go-digest v1.0.0 // indirect
	github.com/opencontainers/image-spec v1.1.0 // indirect
	github.com/opencontainers/runc v1.1.14 // indirect
	github.com/pion/datachannel v1.5.9 // indirect
	github.com/pion/logging v0.2.2 // indirect
	github.com/pion/mdns v0.0.12 // indirect
	github.com/pion/randutil v0.1.0 // indirect
	github.com/pion/srtp/v2 v2.0.20 // indirect
	github.com/pion/stun v0.6.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/prometheus/client_model v0.6.1 // indirect
	github.com/prometheus/common v0.55.0 // indirect
	github.com/prometheus/procfs v0.15.1 // indirect
	github.com/puzpuzpuz/xsync/v3 v3.1.0 // indirect
	github.com/russross/blackfriday/v2 v2.1.0 // indirect
	github.com/sirupsen/logrus v1.9.3 // indirect
	github.com/stoewer/go-strcase v1.3.0 // indirect
	github.com/wlynxg/anet v0.0.5 // indirect
	github.com/xeipuuv/gojsonpointer v0.0.0-20190905194746-02993c407bfb // indirect
	github.com/xeipuuv/gojsonreference v0.0.0-20180127040603-bd5ef7bd5415 // indirect
	github.com/xeipuuv/gojsonschema v1.2.0 // indirect
	github.com/xrash/smetrics v0.0.0-20240521201337-686a1a2994c1 // indirect
	github.com/zeebo/xxh3 v1.0.2 // indirect
	go.uber.org/zap/exp v0.3.0 // indirect
	golang.org/x/crypto v0.28.0 // indirect
	golang.org/x/mod v0.21.0 // indirect
	golang.org/x/net v0.30.0 // indirect
	golang.org/x/sys v0.26.0 // indirect
	golang.org/x/text v0.19.0 // indirect
	golang.org/x/tools v0.26.0 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20240814211410-ddb44dafa142 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20241015192408-796eee8c2d53 // indirect
	google.golang.org/grpc v1.67.1 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
)
