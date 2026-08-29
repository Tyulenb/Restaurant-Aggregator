package establishment

// MenuResponse is the customer-facing menu of an establishment.
type MenuResponse struct {
	Establishment Establishment  `json:"establishment"`
	Categories    []MenuCategory `json:"categories"`
}

// Establishment contains the establishment details displayed with its menu.
type Establishment struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Description *string `json:"description"`
	Available   bool    `json:"available"`
}

// MenuCategory is an active category and its active products.
type MenuCategory struct {
	ID          string        `json:"id"`
	ExternalID  string        `json:"external_id"`
	Name        string        `json:"name"`
	Description *string       `json:"description"`
	Products    []MenuProduct `json:"products"`
}

// MenuProduct is the customer-facing representation of a product.
type MenuProduct struct {
	ID            string  `json:"id"`
	ExternalID    string  `json:"external_id"`
	Name          string  `json:"name"`
	Description   *string `json:"description"`
	Price         float32 `json:"price"`
	Available     bool    `json:"available"`
	InventoryType string  `json:"inventory_type"`
	Quantity      *int    `json:"quantity"`
}
