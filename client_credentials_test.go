package courier

import (
	"context"
	"testing"

	"github.com/stretchr/testify/mock"
)

type mockCredentialFetcher struct {
	mock.Mock
}

func newMockCredentialFetcher(t *testing.T) *mockCredentialFetcher {
	m := &mockCredentialFetcher{}
	m.Test(t)
	return m
}

func (m *mockCredentialFetcher) Credentials(ctx context.Context) (*Credential, error) {
	args := m.Called(ctx)
	if c := args.Get(0); c != nil {
		return c.(*Credential), args.Error(1)
	}
	return nil, args.Error(1)
}
