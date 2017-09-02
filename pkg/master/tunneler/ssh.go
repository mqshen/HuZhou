package tunneler

import "net"

type InstallSSHKey func(user string, data []byte) error

type AddressFunc func() (addresses []string, err error)

type Tunneler interface {
	Run(AddressFunc)
	Stop()
	Dial(net, addr string) (net.Conn, error)
	SecondsSinceSync() int64
	SecondsSinceSSHKeySync() int64
}