package logger_test

import (
	"context"
	"testing"

	"github.com/rotationalio/exchequer/pkg/logger"
	"github.com/rotationalio/exchequer/pkg/ulids"

	"github.com/stretchr/testify/require"
)

func TestRequestIDContext(t *testing.T) {
	requestID := ulids.New().String()
	parent, cancel := context.WithCancel(context.Background())
	ctx := logger.WithRequestID(parent, requestID)

	cmp, ok := logger.RequestID(ctx)
	require.True(t, ok)
	require.Equal(t, requestID, cmp)

	cancel()
	require.ErrorIs(t, ctx.Err(), context.Canceled)
}
