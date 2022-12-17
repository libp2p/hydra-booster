package metrics

import (
	"context"
	"errors"
	"strconv"

	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
	rcmgr "github.com/libp2p/go-libp2p/p2p/host/resource-manager"
	"go.opencensus.io/stats"
	"go.opencensus.io/tag"
)

func CreateRcmgrMetrics(ctx context.Context) (rcmgr.MetricsReporter, error) {
	name, ok := tag.FromContext(ctx).Value(KeyName)
	if !ok {
		return nil, errors.New("context must contain a 'name' key")
	}
	return rcmgrMetrics{name: name}, nil
}

type rcmgrMetrics struct {
	name string
}

func getDirection(d network.Direction) string {
	switch d {
	default:
		return ""
	case network.DirInbound:
		return "inbound"
	case network.DirOutbound:
		return "outbound"
	}
}

func (r rcmgrMetrics) AllowConn(dir network.Direction, usefd bool) {
	stats.RecordWithTags(
		context.Background(),
		[]tag.Mutator{
			tag.Upsert(KeyName, r.name),
			tag.Upsert(KeyDirection, getDirection(dir)),
			tag.Upsert(KeyUsesFD, strconv.FormatBool(usefd)),
		},
		RcmgrConnsAllowed.M(1),
		RcmgrConnsBlocked.M(0),
	)
}

func (r rcmgrMetrics) BlockConn(dir network.Direction, usefd bool) {
	stats.RecordWithTags(
		context.Background(),
		[]tag.Mutator{
			tag.Upsert(KeyName, r.name),
			tag.Upsert(KeyDirection, getDirection(dir)),
			tag.Update(KeyUsesFD, strconv.FormatBool(usefd)),
		},
		RcmgrConnsAllowed.M(0),
		RcmgrConnsBlocked.M(1),
	)
}

func (r rcmgrMetrics) AllowStream(_ peer.ID, dir network.Direction) {
	stats.RecordWithTags(
		context.Background(),
		[]tag.Mutator{
			tag.Upsert(KeyName, r.name),
			tag.Upsert(KeyDirection, getDirection(dir)),
		},
		RcmgrStreamsAllowed.M(1),
		RcmgrStreamsBlocked.M(0),
	)
}

func (r rcmgrMetrics) BlockStream(_ peer.ID, dir network.Direction) {
	stats.RecordWithTags(
		context.Background(),
		[]tag.Mutator{
			tag.Upsert(KeyName, r.name),
			tag.Upsert(KeyDirection, getDirection(dir)),
		},
		RcmgrStreamsAllowed.M(0),
		RcmgrStreamsBlocked.M(1),
	)
}

func (r rcmgrMetrics) AllowPeer(_ peer.ID) {
	stats.RecordWithTags(
		context.Background(),
		[]tag.Mutator{
			tag.Upsert(KeyName, r.name),
		},
		RcmgrPeersAllowed.M(1),
		RcmgrPeersBlocked.M(0),
	)
}

func (r rcmgrMetrics) BlockPeer(_ peer.ID) {
	stats.RecordWithTags(
		context.Background(),
		[]tag.Mutator{
			tag.Upsert(KeyName, r.name),
		},
		RcmgrPeersAllowed.M(0),
		RcmgrPeersBlocked.M(1),
	)
}

func (r rcmgrMetrics) AllowProtocol(proto protocol.ID) {
	stats.RecordWithTags(
		context.Background(),
		[]tag.Mutator{
			tag.Upsert(KeyName, r.name),
			tag.Upsert(KeyProtocol, string(proto)),
		},
		RcmgrProtocolsAllowed.M(1),
		RcmgrProtocolsBlocked.M(0),
	)
}

func (r rcmgrMetrics) BlockProtocol(proto protocol.ID) {
	stats.RecordWithTags(
		context.Background(),
		[]tag.Mutator{
			tag.Upsert(KeyName, r.name),
			tag.Upsert(KeyProtocol, string(proto)),
		},
		RcmgrProtocolsAllowed.M(0),
		RcmgrProtocolsBlocked.M(1),
	)
}

func (r rcmgrMetrics) BlockProtocolPeer(proto protocol.ID, _ peer.ID) {
	stats.RecordWithTags(
		context.Background(),
		[]tag.Mutator{
			tag.Upsert(KeyName, r.name),
			tag.Upsert(KeyProtocol, string(proto)),
		},
		RcmgrProtocolPeersBlocked.M(1),
	)
}

func (r rcmgrMetrics) AllowService(svc string) {
	stats.RecordWithTags(
		context.Background(),
		[]tag.Mutator{
			tag.Upsert(KeyName, r.name),
			tag.Upsert(KeyService, svc),
		},
		RcmgrServiceAllowed.M(1),
		RcmgrServiceBlocked.M(0),
	)
}

func (r rcmgrMetrics) BlockService(svc string) {
	stats.RecordWithTags(
		context.Background(),
		[]tag.Mutator{
			tag.Upsert(KeyName, r.name),
			tag.Upsert(KeyService, svc),
		},
		RcmgrServiceAllowed.M(0),
		RcmgrServiceBlocked.M(1),
	)
}

func (r rcmgrMetrics) BlockServicePeer(svc string, _ peer.ID) {
	stats.RecordWithTags(
		context.Background(),
		[]tag.Mutator{
			tag.Upsert(KeyName, r.name),
			tag.Upsert(KeyService, svc),
		},
		RcmgrServicePeersBlocked.M(1),
	)
}

func (r rcmgrMetrics) AllowMemory(_ int) {
	stats.RecordWithTags(
		context.Background(),
		[]tag.Mutator{
			tag.Upsert(KeyName, r.name),
		},
		RcmgrMemoryAllowed.M(1),
		RcmgrMemoryBlocked.M(0),
	)
}

func (r rcmgrMetrics) BlockMemory(_ int) {
	stats.RecordWithTags(
		context.Background(),
		[]tag.Mutator{
			tag.Upsert(KeyName, r.name),
		},
		RcmgrMemoryAllowed.M(0),
		RcmgrMemoryBlocked.M(1),
	)
}
