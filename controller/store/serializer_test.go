package store

import (
	"fmt"
	"net/netip"
	"testing"

	"github.com/caldog20/overlay/controller/types"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/schema"
)

func TestSerializers(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file::memory:?cache=shared&_journal_mode=WAL")), &gorm.Config{
		PrepareStmt: true, Logger: logger.Default.LogMode(logger.Error),
	})
	if err != nil {
		t.Fatal(err)
	}

	schema.RegisterSerializer("addr", AddrSerializer{})
	schema.RegisterSerializer("addrport", AddrPortSerializer{})

	err = db.AutoMigrate(&types.Peer{})
	if err != nil {
		t.Fatal(err)
	}

	store := &Store{db: db}

	addr := netip.MustParseAddr("100.70.100.1")
	addrPort := netip.MustParseAddrPort("1.2.3.4:5443")

	peer := &types.Peer{
		ID:        0,
		PublicKey: "key",
		IP:        addr,
		Endpoint:  addrPort,
		Connected: false,
	}

	err = store.CreatePeer(peer)
	if err != nil {
		t.Fatal(err)
	}

	dbpeer, err := store.GetPeerByKey("key")
	if err != nil {
		t.Fatal(err)
	}

	assert.EqualValues(t, addr, dbpeer.IP, "peer addr doesn't match dbpeer addr")
	assert.EqualValues(t, addrPort, dbpeer.Endpoint, "peer addr/port doesn't match dbpeer addr/port")

	//fmt.Printf("%v", dbpeer)
}
