package fpnn

type Logger interface {
	Println(...any)
	Printf(string, ...any)
}
