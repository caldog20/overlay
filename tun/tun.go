package tun

const MTU = 1300

type Tun interface {
	Read(b []byte) (int, error)
	Write(b []byte) (int, error)

	Name() (string, error)
	Close() error
	MTU() (int, error)
}
