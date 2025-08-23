package service

type Shortener interface {
	Shorten(string) (string, error)
}
