package sender

type Sender interface {
	Send(message string) error
}
