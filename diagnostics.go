package main

import (
	"bufio"
	"fmt"
	"net"
	"unicode/utf8"

	log "github.com/sirupsen/logrus"
)

func acceptTCPMessage(ln *net.TCPListener, message string, c chan bool) {
	log.WithFields(log.Fields{
		"address": ln.Addr(),
	}).Info("Trying to accept a connection on a TCP socket")
	conn, err := ln.AcceptTCP()
	if err != nil {
		log.WithFields(log.Fields{
			"address": ln.Addr(),
			"error":   err.Error(),
		}).Error("Error accepting a connection on a TCP socket")
		c <- false
		return
	}
	log.WithFields(log.Fields{
		"localAddress":  conn.LocalAddr(),
		"remoteAddress": conn.RemoteAddr(),
	}).Info("Accepted a new TCP connection")
	defer conn.Close()

	log.WithFields(log.Fields{
		"localAddress":  conn.LocalAddr(),
		"remoteAddress": conn.RemoteAddr(),
	}).Info("Trying to read a line from the TCP connection")
	buf, isPrefix, err := bufio.NewReaderSize(conn, 512).ReadLine()
	if err != nil {
		log.WithFields(log.Fields{
			"localAddress":  conn.LocalAddr(),
			"remoteAddress": conn.RemoteAddr(),
		}).Error("Error reading from the TCP connection")
		c <- false
		return
	} else if !utf8.Valid(buf) {
		log.WithFields(log.Fields{
			"localAddress":  conn.LocalAddr(),
			"remoteAddress": conn.RemoteAddr(),
		}).Error("Non-UTF-8 bytes received from the TCP connection")
		c <- false
		return
	} else if isPrefix {
		log.WithFields(log.Fields{
			"localAddress":  conn.LocalAddr(),
			"remoteAddress": conn.RemoteAddr(),
			"message":       string(buf),
		}).Error("Received message didn't fit in the input buffer")
		c <- false
		return
	} else if string(buf) != message {
		log.WithFields(log.Fields{
			"localAddress":  conn.LocalAddr(),
			"remoteAddress": conn.RemoteAddr(),
			"expected":      message,
			"message":       string(buf),
		}).Error("Received message did not match the expected message")
		c <- false
		return
	}

	c <- true
}

func sendTCPMessage(network, addr, message string) (conn net.Conn, err error) {
	log.WithFields(log.Fields{
		"network": network,
		"address": addr,
	}).Info("Trying to connect to a TCP socket")
	conn, err = net.Dial(network, addr)
	if err != nil {
		log.WithFields(log.Fields{
			"network": network,
			"address": addr,
		}).Error("Error connecting to a TCP socket")
		return
	}

	log.WithFields(log.Fields{
		"network": network,
		"address": addr,
		"message": message,
	}).Info("Writing a message to a TCP socket")
	_, err = fmt.Fprintf(conn, "%s\n", message)
	if err != nil {
		log.WithFields(log.Fields{
			"network": network,
			"address": addr,
			"message": message,
		}).Error("Error writing to a TCP socket")
		return
	}
	log.WithFields(log.Fields{
		"network": network,
		"address": addr,
		"message": message,
	}).Info("Wrote a message to a TCP socket successfully")
	return
}

func testTCPConnectivity(network, publicAddr, privateAddr string) bool {
	privateTCPAddr, err := net.ResolveTCPAddr("tcp", privateAddr)
	if err != nil {
		log.WithFields(log.Fields{
			"address": privateAddr,
			"error":   err.Error(),
		}).Error("Error resolving a TCP address")
		return false
	}

	log.WithFields(log.Fields{
		"address": privateTCPAddr.String(),
	}).Info("Trying to listen to a TCP socket")
	ln, err := net.ListenTCP("tcp", privateTCPAddr)
	if err != nil {
		log.WithFields(log.Fields{
			"error": err.Error(),
		}).Error("Error listening to a TCP socket")
		return false
	}
	defer ln.Close()

	ch := make(chan bool)
	go acceptTCPMessage(ln, privateAddr, ch)

	success := false
	conn, err := sendTCPMessage(network, publicAddr, privateAddr)
	if conn != nil {
		defer conn.Close()
	}
	if err == nil {
		success = <-ch
	}

	if success {
		log.WithFields(log.Fields{
			"network": network,
			"address": publicAddr,
		}).Info("Connecting to a TCP socket was successful")
	} else {
		log.WithFields(log.Fields{
			"network": network,
			"address": publicAddr,
		}).Warning("Connecting to a TCP socket failed")
	}

	return success
}
