package providers

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/benbjohnson/clock"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-multiaddr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

var tableName = "testtable"

func startDDBLocal(ctx context.Context, ddbClient *dynamodb.Client) (func(), error) {
	cmd := exec.Command("docker", "run", "-d", "-p", "8000:8000", "amazon/dynamodb-local", "-jar", "DynamoDBLocal.jar", "-inMemory")
	buf := &bytes.Buffer{}
	cmd.Stdout = buf
	cmd.Stderr = buf
	err := cmd.Run()
	if err != nil {
		return nil, fmt.Errorf("error running DynamoDB Local (%s), output:\n%s", err.Error(), buf)
	}

	ctrID := strings.TrimSpace(buf.String())

	cleanupFunc := func() {
		cmd := exec.Command("docker", "kill", ctrID)
		if err := cmd.Run(); err != nil {
			fmt.Printf("error killing %s: %s\n", ctrID, err)
		}
	}

	// wait for DynamoDB to respond
	for {
		select {
		case <-ctx.Done():
			cleanupFunc()
			return nil, ctx.Err()
		default:
		}

		_, err := ddbClient.ListTables(ctx, &dynamodb.ListTablesInput{})
		if err == nil {
			break
		}
	}

	return cleanupFunc, err
}

func newDDBClient() *dynamodb.Client {
	resolver := dynamodb.EndpointResolverFunc(func(region string, options dynamodb.EndpointResolverOptions) (aws.Endpoint, error) {
		return aws.Endpoint{
			PartitionID:   "aws",
			URL:           "http://localhost:8000",
			SigningRegion: region,
		}, nil
	})
	cfg, err := config.LoadDefaultConfig(context.Background(),
		config.WithRegion("us-east-1"),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider("a", "a", "a")),
	)
	if err != nil {
		panic(err)
	}
	return dynamodb.NewFromConfig(cfg, dynamodb.WithEndpointResolver(resolver))
}

func setupTables(ddbClient *dynamodb.Client) error {
	_, err := ddbClient.CreateTable(context.Background(), &dynamodb.CreateTableInput{
		AttributeDefinitions: []types.AttributeDefinition{
			{
				AttributeName: aws.String("key"),
				AttributeType: types.ScalarAttributeTypeB,
			},
			{
				AttributeName: aws.String("ttl"),
				AttributeType: types.ScalarAttributeTypeN,
			},
		},
		KeySchema: []types.KeySchemaElement{
			{
				AttributeName: aws.String("key"),
				KeyType:       types.KeyTypeHash,
			},
			{
				AttributeName: aws.String("ttl"),
				KeyType:       types.KeyTypeRange,
			},
		},
		TableName:   &tableName,
		BillingMode: types.BillingModeProvisioned,
		ProvisionedThroughput: &types.ProvisionedThroughput{
			ReadCapacityUnits:  aws.Int64(1000),
			WriteCapacityUnits: aws.Int64(1000),
		},
	})
	if err != nil {
		return err
	}
	_, err = ddbClient.UpdateTimeToLive(context.Background(), &dynamodb.UpdateTimeToLiveInput{
		TableName: &tableName,
		TimeToLiveSpecification: &types.TimeToLiveSpecification{
			AttributeName: aws.String("ttl"),
			Enabled:       aws.Bool(true),
		},
	})
	return err
}

type mockPeerStore struct {
	addrs map[string][]multiaddr.Multiaddr
}

func (m *mockPeerStore) PeerInfo(peerID peer.ID) peer.AddrInfo {
	return peer.AddrInfo{
		ID:    peerID,
		Addrs: m.addrs[string(peerID)],
	}
}
func (m *mockPeerStore) AddAddrs(p peer.ID, addrs []multiaddr.Multiaddr, ttl time.Duration) {
	m.addrs[string(p)] = append(m.addrs[string(p)], addrs...)
}

func TestProviderStore_ddb_local(t *testing.T) {
	ctx, stop := context.WithTimeout(context.Background(), 300*time.Second)
	defer stop()

	ddb := newDDBClient()
	stopDDBLocal, err := startDDBLocal(ctx, ddb)
	assert.NoError(t, err)
	t.Cleanup(stopDDBLocal)

	err = setupTables(ddb)
	assert.NoError(t, err)

	mockClock := clock.NewMock()
	peerStore := &mockPeerStore{addrs: map[string][]multiaddr.Multiaddr{}}
	provStore := &dynamoDBProviderStore{
		Self:       "peer",
		Peerstore:  peerStore,
		DDBClient:  ddb,
		TableName:  tableName,
		TTL:        100 * time.Second,
		QueryLimit: 10,
		clock:      mockClock,
	}

	key := []byte("foo")
	ma, err := multiaddr.NewMultiaddr("/ip4/1.1.1.1")
	assert.NoError(t, err)

	// add more providers than the query limit to ensure the limit is enforced
	numProvs := int(provStore.QueryLimit * 2)
	for i := 0; i < numProvs; i++ {
		peerID := i
		prov := peer.AddrInfo{
			ID:    peer.ID(strconv.Itoa(peerID)),
			Addrs: []multiaddr.Multiaddr{ma},
		}
		err = provStore.AddProvider(ctx, key, prov)
		if err != nil {
			t.Fatal(err)
		}
		mockClock.Add(1 * time.Second)
	}

	provs, err := provStore.GetProviders(ctx, key)
	assert.NoError(t, err)
	assert.EqualValues(t, provStore.QueryLimit, len(provs))

	for i, prov := range provs {
		// peer ids should be decreasing since results should be sorted by most-recently-added-first
		assert.NoError(t, err)
		expID := strconv.Itoa(numProvs - i - 1)
		assert.Equal(t, expID, string(prov.ID))

		assert.Len(t, prov.Addrs, 1)
		assert.True(t, ma.Equal(prov.Addrs[0]))
	}

	n, err := provStore.CountProviderRecords(ctx)
	assert.NoError(t, err)
	assert.EqualValues(t, numProvs, n)
}

type mockDDB struct{ mock.Mock }

// note that variadic args don't work on these mocks but we don't use them anyway
func (m *mockDDB) Query(ctx context.Context, params *dynamodb.QueryInput, optFns ...func(*dynamodb.Options)) (*dynamodb.QueryOutput, error) {
	args := m.Called(ctx, params, optFns)
	return args.Get(0).(*dynamodb.QueryOutput), args.Error(1)
}

func (m *mockDDB) DescribeTable(ctx context.Context, params *dynamodb.DescribeTableInput, optFns ...func(*dynamodb.Options)) (*dynamodb.DescribeTableOutput, error) {
	args := m.Called(ctx, params, optFns)
	return args.Get(0).(*dynamodb.DescribeTableOutput), args.Error(1)
}

func (m *mockDDB) PutItem(ctx context.Context, params *dynamodb.PutItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error) {
	args := m.Called(ctx, params, optFns)
	return args.Get(0).(*dynamodb.PutItemOutput), args.Error(1)
}

func TestProviderStore_pagination(t *testing.T) {
	// we can't use DynamoDB Local for this
	// because we need to make the response page size much smaller to exercise pagination
	ctx, stop := context.WithTimeout(context.Background(), 1*time.Second)
	defer stop()
	ddbClient := &mockDDB{}
	mockClock := clock.NewMock()
	peerStore := &mockPeerStore{addrs: map[string][]multiaddr.Multiaddr{}}
	provStore := &dynamoDBProviderStore{
		Self:       "peer",
		Peerstore:  peerStore,
		DDBClient:  ddbClient,
		TableName:  tableName,
		TTL:        100 * time.Second,
		QueryLimit: 10,
		// set page limit low so we exercise pagination
		clock: mockClock,
	}

	// return 2 pages to exercise pagination logic

	ddbClient.
		On("Query", ctx, mock.Anything, mock.Anything).
		Return(&dynamodb.QueryOutput{
			Items: []map[string]types.AttributeValue{
				{"prov": &types.AttributeValueMemberB{Value: []byte("1")}},
			},
			LastEvaluatedKey: map[string]types.AttributeValue{
				"prov": &types.AttributeValueMemberB{Value: []byte("1")},
			},
		}, nil).
		Times(1)
	ddbClient.
		On("Query", ctx, mock.Anything, mock.Anything).
		Return(&dynamodb.QueryOutput{
			Items: []map[string]types.AttributeValue{
				{"prov": &types.AttributeValueMemberB{Value: []byte("2")}},
			},
		}, nil).
		Times(1)

	provs, err := provStore.GetProviders(ctx, []byte("key"))
	assert.NoError(t, err)
	assert.EqualValues(t, 2, len(provs))
	assert.EqualValues(t, "1", provs[0].ID)
	assert.EqualValues(t, "2", provs[1].ID)
}

func TestProviderStore_pagination_no_results(t *testing.T) {
	// we can't use DynamoDB Local for this
	// because we need to make the response page size much smaller to exercise pagination
	ctx, stop := context.WithTimeout(context.Background(), 1*time.Second)
	defer stop()
	ddbClient := &mockDDB{}
	mockClock := clock.NewMock()
	peerStore := &mockPeerStore{addrs: map[string][]multiaddr.Multiaddr{}}
	provStore := &dynamoDBProviderStore{
		Self:       "peer",
		Peerstore:  peerStore,
		DDBClient:  ddbClient,
		TableName:  tableName,
		TTL:        100 * time.Second,
		QueryLimit: 10,
		// set page limit low so we exercise pagination
		clock: mockClock,
	}

	// return 2 pages to exercise pagination logic

	ddbClient.
		On("Query", ctx, mock.Anything, mock.Anything).
		Return(&dynamodb.QueryOutput{}, nil).
		Times(1)

	provs, err := provStore.GetProviders(ctx, []byte("key"))
	assert.NoError(t, err)
	assert.EqualValues(t, 0, len(provs))
}
