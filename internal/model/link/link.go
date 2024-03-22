package link

type Link struct {
	Url        string
	Lang       string
	ResourceID int
}

func NewLink(url string, lang string, resourceID int) *Link {
	return &Link{
		Url:        url,
		Lang:       lang,
		ResourceID: resourceID,
	}
}
