package providers

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/benbjohnson/clock"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/peerstore"
	"github.com/libp2p/hydra-booster/metrics"
	"github.com/multiformats/go-multiaddr"
	"go.opencensus.io/stats"
)

type ddbClient interface {
	dynamodb.QueryAPIClient
	dynamodb.DescribeTableAPIClient
	PutItem(ctx context.Context, params *dynamodb.PutItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error)
}

type peerStore interface {
	PeerInfo(peer.ID) peer.AddrInfo
	AddAddrs(p peer.ID, addrs []multiaddr.Multiaddr, ttl time.Duration)
}

type dynamoDBProviderStore struct {
	Self       peer.ID
	Peerstore  peerStore
	DDBClient  ddbClient
	TableName  string
	TTL        time.Duration
	QueryLimit int32
	clock      clock.Clock
}

func NewDynamoDBProviderStore(self peer.ID, peerstore peerStore, ddbClient ddbClient, tableName string, ttl time.Duration, queryLimit int32) *dynamoDBProviderStore {
	return &dynamoDBProviderStore{
		Self:       "peer",
		Peerstore:  peerstore,
		DDBClient:  ddbClient,
		TableName:  tableName,
		TTL:        ttl,
		QueryLimit: queryLimit,
		clock:      clock.New(),
	}
}

func (d *dynamoDBProviderStore) AddProvider(ctx context.Context, key []byte, prov peer.AddrInfo) error {
	if prov.ID != d.Self { // don't add own addrs.
		d.Peerstore.AddAddrs(prov.ID, prov.Addrs, peerstore.ProviderAddrTTL)
	}

	ttlEpoch := d.clock.Now().Add(d.TTL).UnixNano() / 1e9
	ttlEpochStr := strconv.FormatInt(ttlEpoch, 10)
	_, err := d.DDBClient.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: &d.TableName,
		Item: map[string]types.AttributeValue{
			"key":  &types.AttributeValueMemberB{Value: key},
			"prov": &types.AttributeValueMemberB{Value: []byte(prov.ID)},
			"ttl":  &types.AttributeValueMemberN{Value: ttlEpochStr},
		},
		ConditionExpression: aws.String("attribute_not_exists(#k) AND attribute_not_exists(#t)"),
		ExpressionAttributeNames: map[string]string{
			"#k": "key",
			"#t": "ttl",
		},
	})
	var ccfe *types.ConditionalCheckFailedException
	if errors.As(err, &ccfe) {
		// the item already exists which means we tried to write >1 providers for a CID at the exact same millisecond
		// nothing to do, move on
		// (there is a metric recorded for this, since all error codes are recorded)
		return nil
	}
	return err
}

func (d *dynamoDBProviderStore) GetProviders(ctx context.Context, key []byte) ([]peer.AddrInfo, error) {
	providersLeft := d.QueryLimit

	// dedupe the providers and preserve order
	providersSet := map[string]bool{}
	providers := []peer.AddrInfo{}

	var startKey map[string]types.AttributeValue
	for {
		res, err := d.DDBClient.Query(ctx, &dynamodb.QueryInput{
			TableName:              &d.TableName,
			KeyConditionExpression: aws.String("#k = :key"),
			ExpressionAttributeValues: map[string]types.AttributeValue{
				":key": &types.AttributeValueMemberB{Value: key},
			},
			ExpressionAttributeNames: map[string]string{
				"#k": "key",
			},
			ScanIndexForward:  aws.Bool(false), // return most recent entries first
			ExclusiveStartKey: startKey,
			Limit:             &providersLeft,
		})
		if err != nil {
			return nil, err
		}
		for _, item := range res.Items {
			prov, ok := item["prov"]
			if !ok {
				return nil, errors.New("unexpected item without a 'prov' attribute")
			}
			provB, ok := prov.(*types.AttributeValueMemberB)
			if !ok {

				return nil, fmt.Errorf("unexpected value type of '%s' for 'prov' attribute", reflect.TypeOf(prov))
			}
			provStr := string(provB.Value)
			peerID := peer.ID(string(provB.Value))
			addrInfo := d.Peerstore.PeerInfo(peerID)

			if _, ok := providersSet[provStr]; !ok {
				providersSet[provStr] = true
				providers = append(providers, addrInfo)
			}
		}

		numItems := int32(len(res.Items))
		if numItems >= providersLeft || len(res.LastEvaluatedKey) == 0 {
			break
		}

		providersLeft -= numItems
	}

	stats.Record(ctx, stats.Measurement(metrics.ProviderRecordsPerKey.M(int64(len(providers)))))

	if len(providers) > 0 {
		recordPrefetches(ctx, "local")
	}

	return providers, nil
}

// CountProviderRecords returns the approximate number of records in the table. This shouldn't be called more often than once every few seconds, as
// DynamoDB may start throttling the requests.
func (d *dynamoDBProviderStore) CountProviderRecords(ctx context.Context) (int64, error) {
	res, err := d.DDBClient.DescribeTable(ctx, &dynamodb.DescribeTableInput{
		TableName: &d.TableName,
	})
	if err != nil {
		return 0, err
	}
	return res.Table.ItemCount, nil
}
