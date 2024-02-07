package internal

type T struct{}

func (*T) Hello(msg string, reply *string) error {
	*reply = "Hello " + msg
	return nil
}
