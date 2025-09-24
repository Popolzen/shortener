package repository

type URLRepository interface {
	Store(shortURL, longURL, id string) error
	Get(longURL, id string) (string, error)
}
