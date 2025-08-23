package repository

type URLRepository interface {
	Store(shortURL string, longURL string) error
	Get(longURL string) (string, error)
}
