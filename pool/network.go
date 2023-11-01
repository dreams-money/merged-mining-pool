package pool

import (
	"bufio"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net"
	"time"
)

const extranonce1Length = 4

var numberOfConnections int

type stratumClient struct {
	ip                    string
	login                 string
	extranonce1           string
	extranonce_subscribed bool
	userAgent             string

	sessionID     int
	connection    net.Conn
	streamEncoder *json.Encoder
}

func (pool *PoolServer) listenForConnections() {
	pool.connectionTimeout = mustParseDuration(pool.config.ConnectionTimeout)

	addr, err := net.ResolveTCPAddr("tcp", ":"+pool.config.Port)
	if err != nil {
		panicOnError(err)
	}

	server, err := net.ListenTCP("tcp", addr)
	panicOnError(err)
	defer server.Close()

	connectionChannel := make(chan struct{})
	for { // Listen for connections
		if numberOfConnections > pool.config.MaxConnections {
			log.Println("Maximum number of connections reached")
			// log.Fatal("Maximum number of connections reached")

			continue
		}

		con, err := server.AcceptTCP()
		if err != nil {
			log.Println(err)
			continue
		}
		con.SetKeepAlive(true)

		ip, _, err := net.SplitHostPort(con.RemoteAddr().String())
		if err != nil {
			log.Println(err)
			continue
		}

		log.Println("New Stratum Connection from: " + ip)

		if isBanned(ip) {
			con.Close()
			continue
		}

		client := &stratumClient{
			ip:          ip,
			extranonce1: uniqueExtranonce(extranonce1Length * 2),
			connection:  con,
		}

		go pool.openNewConnection(client, connectionChannel)

		// TODO - do channels need closing?
		<-connectionChannel

		numberOfConnections++
	}
}

const maxRequestSize = 1024

func (pool *PoolServer) openNewConnection(client *stratumClient, connectionChannel chan struct{}) {
	err := pool.handleStratumConnection(client)
	if err != nil {
		log.Println(err)
		removeSession(client.sessionID)
		client.connection.Close()
		numberOfConnections--
	}
	connectionChannel <- struct{}{}
}

func (pool *PoolServer) handleStratumConnection(client *stratumClient) error {
	client.streamEncoder = json.NewEncoder(client.connection)
	connectionBuffer := bufio.NewReaderSize(client.connection, maxRequestSize)

	timeoutTime := time.Now().Add(pool.connectionTimeout)
	client.connection.SetDeadline(timeoutTime)

	for {
		payload, isPrefix, err := connectionBuffer.ReadLine()
		if err == io.EOF {
			removeSession(client.sessionID)
			return errors.New("Client disconnect: " + client.ip)
		}

		if isPrefix {
			log.Println("Socket flood detected from: " + client.ip)
			banClient(client)
			return err
		} else if err != nil {
			log.Println("Socket read error from: " + client.ip)
			return err
		}

		if len(payload) > 1 {
			err = pool.respondToStratumClient(client, payload)
			if err != nil {
				return err
			}
		}
	}
}

func sendPacket(packet any, client *stratumClient) error {
	return client.streamEncoder.Encode(packet)
}

func mustParseDuration(s string) time.Duration {
	value, err := time.ParseDuration(s)
	if err != nil {
		panic("util: Can't parse duration `" + s + "`: " + err.Error())
	}
	return value
}
