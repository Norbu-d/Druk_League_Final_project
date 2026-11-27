package domain

type Hit struct {
	EntityType string
	EntityID   string
	Title      string
	Subtitle   string
}

type Page struct {
	Hits          []Hit
	NextPageToken string
}

type SearchRequest struct {
	Query       string
	EntityTypes []string
	PageSize    int
	PageToken   string
}
