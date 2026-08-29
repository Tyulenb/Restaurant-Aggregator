package adapterhttp

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/talense-tasks/backend-trainee-assignment-autumn-2026-tyulenb-7224f74f/internal/menu/domain"
)

func TestProductToDTO_UsesCategoryExternalID(t *testing.T) {
	product := domain.Product{
		CategoryID:         "16431898-3caf-4a0b-8e47-c34382bc7999",
		CategoryExternalID: "net-cat-1788023629395571143",
	}

	payload, err := json.Marshal(productToDTO(product))
	require.NoError(t, err)

	var response map[string]any
	require.NoError(t, json.Unmarshal(payload, &response))
	assert.Equal(t, product.CategoryExternalID, response["category_external_id"])
	assert.NotContains(t, response, "category_id")
}
