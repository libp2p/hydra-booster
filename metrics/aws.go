package metrics

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	awsmiddle "github.com/aws/aws-sdk-go-v2/aws/middleware"
	"github.com/aws/aws-sdk-go-v2/aws/retry"
	"github.com/aws/smithy-go"
	smithymiddle "github.com/aws/smithy-go/middleware"
	smithyhttp "github.com/aws/smithy-go/transport/http"
	"go.opencensus.io/stats"
	"go.opencensus.io/tag"
)

// AWS SDK middleware that records client side metrics of all AWS SDK requests.
type AWSMetricsMiddleware struct{}

func (m *AWSMetricsMiddleware) ID() string { return "AWSMetrics" }
func (m *AWSMetricsMiddleware) HandleFinalize(ctx context.Context, in smithymiddle.FinalizeInput, next smithymiddle.FinalizeHandler) (smithymiddle.FinalizeOutput, smithymiddle.Metadata, error) {
	start := time.Now()

	out, md, err := next.HandleFinalize(ctx, in)

	service, operation := awsmiddle.GetServiceID(ctx), awsmiddle.GetOperationName(ctx)

	httpCode := 0
	if httpResp, ok := awsmiddle.GetRawResponse(md).(*smithyhttp.Response); ok {
		httpCode = httpResp.StatusCode
	}
	errCode := "none"
	var apiErr smithy.APIError
	if errors.As(err, &apiErr) {
		errCode = apiErr.ErrorCode()
	}
	tags := []tag.Mutator{
		tag.Upsert(KeyOperation, fmt.Sprintf("%s.%s", service, operation)),
		tag.Upsert(KeyHTTPCode, strconv.Itoa(httpCode)),
		tag.Upsert(KeyErrorCode, errCode),
	}

	t, ok := awsmiddle.GetResponseAt(md)
	if ok {
		stats.RecordWithTags(ctx, tags, AWSRequestDurationMillis.M(float64(t.Sub(start).Milliseconds())))
	}

	attemptResults, ok := retry.GetAttemptResults(md)
	if ok {
		retries := int64(0)
		for _, result := range attemptResults.Results {
			if result.Retried {
				retries++
			}
		}
		stats.RecordWithTags(ctx, tags, AWSRequests.M(int64(len(attemptResults.Results))))
		stats.RecordWithTags(ctx, tags, AWSRequestRetries.M(retries))
	}

	return out, md, err
}

var _ smithymiddle.FinalizeMiddleware = (*AWSMetricsMiddleware)(nil)

// Install the AWS metrics middleware onto an SDK stack.
// Typically you use this on a config object like this:
//
//	awsCfg.APIOptions = append(awsCfg.APIOptions, metrics.AddAWSSDKMiddleware)
func AddAWSSDKMiddleware(stack *smithymiddle.Stack) error {
	m := &AWSMetricsMiddleware{}
	return stack.Finalize.Add(m, smithymiddle.Before)
}
