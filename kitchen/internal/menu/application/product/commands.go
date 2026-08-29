package product

type CommandCreateProduct struct {
	ExternalID         string
	CategoryExternalID string
	Name               string
	Description        *string
	Price              float32
	Available          bool
	InventoryType      string
	Quantity           *int
}

type CommandUpdateProduct struct {
	CategoryExternalID *string
	Name               *string
	Description        *string
	DescriptionSet     bool
	Price              *float32
	Available          *bool
}
