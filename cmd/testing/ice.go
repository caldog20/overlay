// SPDX-FileCopyrightText: 2023 The Pion community <https://pion.ly>
// SPDX-License-Identifier: MIT

// Package main implements a simple example demonstrating a Pion-to-Pion ICE connection
package main

import (
	"net"

	"github.com/pion/ice/v3"
)

func main() { //nolint
	conn, _ := net.ListenUDP("udp4", &net.UDPAddr{Port: 5000})

	mux := ice.NewUniversalUDPMuxDefault(ice.UniversalUDPMuxParams{Logger: nil, UDPConn: conn})

	defer mux.Close()
	defer conn.Close()

}
