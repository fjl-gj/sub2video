package cos

import "context"

func (c *Client) Provider() string { return "cos" }

func (c *Client) EnsureBucket(ctx context.Context) error {
	_ = ctx
	if !c.Configured() {
		return errNotConfigured{}
	}
	return nil
}

type errNotConfigured struct{}

func (errNotConfigured) Error() string { return "cos not configured" }
