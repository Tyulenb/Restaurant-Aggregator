package adapterhttp

type Establishment struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Description *string `json:"description"`
	Available   bool    `json:"available"`
}

type ChangeAvailabilityRequest struct {
	Available bool `json:"available"`
}
