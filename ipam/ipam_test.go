package ipam

import (
	"net/netip"
)

type ipamtest struct {
	*Ipam
	expectedPrefix netip.Prefix
}

var i = ipamtest{
	Ipam:           &Ipam{},
	expectedPrefix: netip.MustParsePrefix("192.168.1.0/24"),
}

//func TestIpam_SetPrefix(t *testing.T) {
//	err := i.SetPrefix("192.168.1.0/24")
//	if err != nil {
//		t.Fatal(err)
//	}
//	assert.Equal(t, i.expectedPrefix, i.prefix)
//}
//
//func TestIpam_NextIP(t *testing.T) {
//	i.SetPrefix("192.168.1.0/24")
//	ip := i.NextIP()
//	assert.Equal(t, "192.168.1.1", ip)
//}
