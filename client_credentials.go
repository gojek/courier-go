package courier

import "context"

// Credential is a <username,password> pair.
type Credential struct {
	Username string
	Password string
}

// CredentialFetcher is an interface that allows to fetch credentials for a client.
type CredentialFetcher interface {
	Credentials(context.Context) (*Credential, error)
}

// WithCredentialFetcher sets the specified CredentialFetcher.
func WithCredentialFetcher(fetcher CredentialFetcher) ClientOption {
	return optionFunc(func(o *clientOptions) {
		o.credentialFetcher = fetcher
	})
}
