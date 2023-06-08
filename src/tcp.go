package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"time"
)

// TcpListen starts the tcp listening.
func (node *Node) TcpListen(peerIPAddress string) {
	listen, err := net.Listen("tcp", peerIPAddress)
	if err != nil {
		log.Panic(err)
	}
	// fmt.Printf("====TCP Listen: Node starts to listen at address %s====\n", node.NodeAddr)
	defer listen.Close()

	for {
		conn, err := listen.Accept()
		if err != nil {
			log.Panic("Listen Accept Error: ", err)
		}
		startTime := time.Time{}
		if DEBUG {
			startTime = time.Now()
		}
		bMsg, err := ioutil.ReadAll(conn)
		if DEBUG {
			if time.Since(startTime) > 1*time.Second {
				fmt.Println("Msg Read costs: ", time.Since(startTime))
			}
			fmt.Println("Receiving from", peerIPAddress)
		}

		if err != nil {
			// log.Panic("Listen Read Error: ", err)
			fmt.Println("Listen Read Error: ", err)
		}
		go node.HandleMsg(bMsg, peerIPAddress)
	}
}

// TcpDial sends the message in the TCP level.
func TcpDial(context []byte, destAddr string) {
	if DEBUG {
		fmt.Println("Sending to", destAddr)
	}

	// TODO: If it's necessary?
	dialTime := 10
	dialFlag := false
	conn, err := net.Dial("tcp", destAddr)
	if err == nil {
		dialFlag = true
	} else {
		for i := 0; i < dialTime; i++ {
			conn, err = net.Dial("tcp", destAddr)
			if err == nil {
				dialFlag = true
				break
			}
		}
	}
	if !dialFlag {
		// log.Panic("Dail TCP Error: ", err)
		fmt.Println("Dail TCP Error: ", err)
		return
	}

	_, err = conn.Write(context)
	if err != nil {
		// log.Panic("Dail Write Error: ", err)
		fmt.Println("Dail Write Error: ", err)
		return
	}

	err = conn.Close()
	if err != nil {
		log.Panic("Dail Close Error: ", err)
	}
}
