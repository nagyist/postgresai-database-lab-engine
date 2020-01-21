/*
2019 © Postgres.ai
*/

package client

import (
	"context"
	"encoding/json"
	"net/http"

	"gitlab.com/postgres-ai/database-lab/pkg/models"

	"github.com/pkg/errors"
)

// ListSnapshots provides a snapshot list.
func (c *Client) ListSnapshots(ctx context.Context) ([]*models.Snapshot, error) {
	u := c.URL("/snapshots")

	request, err := http.NewRequest(http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to make a request")
	}

	response, err := c.Do(ctx, request)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get response")
	}

	defer func() { _ = response.Body.Close() }()

	var snapshots []*models.Snapshot

	if err := json.NewDecoder(response.Body).Decode(&snapshots); err != nil {
		return nil, errors.Wrap(err, "failed to get response")
	}

	return snapshots, nil
}
