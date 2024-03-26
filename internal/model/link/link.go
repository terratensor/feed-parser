package link

type Link struct {
	Url        string
	Lang       string
	ResourceID int
	UserAgent  string
}

func NewLink(url string, lang string, resourceID int, userAgent string) *Link {
	return &Link{
		Url:        url,
		Lang:       lang,
		ResourceID: resourceID,
		UserAgent:  userAgent,
	}
}
