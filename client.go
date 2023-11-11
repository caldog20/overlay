package main

import (
	"github.com/caldog20/go-overlay/msg"
	"github.com/google/uuid"
)

type Client struct {
	Id uuid.UUID
	//User        string
	TunIP       string
	Remote      string
	PunchStream msg.ControlService_PunchNotifierServer
	Finished    chan<- bool
}
