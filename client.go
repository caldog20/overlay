package main

import "github.com/google/uuid"
import "github.com/caldog20/go-overlay/msg"

type Client struct {
	Id          uuid.UUID
	User        string
	TunIP       string
	Remote      string
	PunchStream msg.ControlService_PunchNotifierServer
	Finished    chan<- bool
}
