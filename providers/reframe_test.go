package providers

import (
	"context"
	"errors"
	"net"
	"reflect"
	"testing"

	"github.com/ipld/edelweiss/services"
	"github.com/stretchr/testify/require"
)

type timeoutNetErr struct{}

func (t *timeoutNetErr) Timeout() bool   { return true }
func (t *timeoutNetErr) Temporary() bool { return false }
func (t *timeoutNetErr) Error() string   { return "timeout" }

func TestMetricsErrStr(t *testing.T) {
	for _, c := range []struct {
		err         error
		expectedStr string
	}{
		{
			err:         context.DeadlineExceeded,
			expectedStr: "DeadlineExceeded",
		},
		{
			err:         context.Canceled,
			expectedStr: "Canceled",
		},
		{
			err:         services.ErrSchema,
			expectedStr: "Schema",
		},
		{
			err:         &services.ErrService{},
			expectedStr: "Service",
		},
		{
			err:         &services.ErrProto{},
			expectedStr: "Proto",
		},
		{
			err:         &net.DNSError{IsNotFound: true},
			expectedStr: "DNSNotFound",
		},
		{
			err:         &net.DNSError{IsTimeout: true},
			expectedStr: "DNSTimeout",
		},
		{
			err:         &timeoutNetErr{},
			expectedStr: "NetTimeout",
		},
		{
			err:         &net.AddrError{},
			expectedStr: "Net",
		},
		{
			err:         errors.New("foo"),
			expectedStr: "Other",
		},
	} {
		errType := reflect.TypeOf(c.err).String()
		t.Run(errType, func(t *testing.T) {
			require.Equal(t, c.expectedStr, metricsErrStr(c.err))
		})
	}
}
