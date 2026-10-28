package consumer

import (
	"testing"

	"go.uber.org/mock/gomock"

	nssf_context "github.com/free5gc/nssf/internal/context"
	"github.com/free5gc/nssf/pkg/app"
	"github.com/free5gc/nssf/pkg/factory"
)

func newTestConsumer(t *testing.T, ctx *nssf_context.NSSFContext) *Consumer {
	t.Helper()

	controller := gomock.NewController(t)
	mockApp := app.NewMockNssfApp(controller)
	mockApp.EXPECT().Context().Return(ctx).AnyTimes()
	// The same pointer on every call: the fallback test mutates the configuration.
	config := &factory.Config{Configuration: &factory.Configuration{}}
	mockApp.EXPECT().Config().Return(config).AnyTimes()

	testConsumer, err := NewConsumer(mockApp)
	if err != nil {
		t.Fatalf("NewConsumer: %v", err)
	}

	return testConsumer
}
