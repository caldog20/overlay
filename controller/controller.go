package controller

import (
	"log"
	"math/rand"
	"net/netip"
	"strings"
	"sync"

	"github.com/caldog20/overlay/controller/store"
	"github.com/caldog20/overlay/controller/types"
)

const (
	Prefix = "100.70.0.0/24"
)

type Controller struct {
	store        *store.Store
	prefix       netip.Prefix
	config       *types.Config
	peerChannels sync.Map
}

func NewController(store *store.Store) *Controller {
	// TODO: Pull settings from config struct
	c := &Controller{
		store:        store,
		peerChannels: sync.Map{},
		prefix:       netip.MustParsePrefix(Prefix),
	}

	return c
}

func (c *Controller) AllocatePeerIP() (netip.Addr, error) {
	usedIPs, err := c.store.GetAllocatedIPs()
	if err != nil {
		return netip.Addr{}, err
	}

	ip := c.prefix.Addr().Next()

	for _, usedIP := range usedIPs {
		addr := netip.MustParseAddr(usedIP)
		if addr.Compare(ip) != 0 {
			break
		}
		ip = ip.Next()
	}

	return ip, nil
}

func (c *Controller) CreateAdminUser() error {
	log.Println("checking if admin user exists")
	tempPass := "admin123"
	admin, err := types.NewUser("admin", tempPass)
	if err != nil {
		log.Printf("error creating admin user: %s", err.Error())
		return err
	}

	err = c.store.CreateUser(admin)
	if err != nil {
		log.Printf("error creating admin user: %s", err.Error())
		return err
	}

	log.Printf("admin user created - password: %s", tempPass)

	return nil
}

func generateRandomPassword() string {
	chars := []rune("ABCDEFGHIJKLMNOPQRSTUVWXYZ" +
		"abcdefghijklmnopqrstuvwxyz" +
		"0123456789")
	length := 12

	var b strings.Builder

	for i := 0; i < length; i++ {
		b.WriteRune(chars[rand.Intn(len(chars))])
	}

	return b.String()
}
