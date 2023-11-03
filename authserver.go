package main

import (
	"database/sql"
	"fmt"
	"log"
	"net"

	"github.com/gookit/config/v2"
)

type AuthServer struct {
	DBInfo DBInfo
	Config ServerConfig
}

type DBInfo struct {
	db     *sql.DB
	DBAddr string
	DBPort string
}

type ServerConfig struct {
	dev                  bool
	listenPort           string
	maxConnections       int
	acceptingConnections bool
}

func (aServer AuthServer) StartServer() error {

	fmt.Println("Starting auth server..")

	conn, err := net.Listen("tcp4", aServer.Config.listenPort)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Auth Server waiting for connections!")

	for {
		connection, err := conn.Accept()
		if err != nil {
			log.Fatal(err)
		}

		go HandleConnection(connection)

	}
}

func main() {

	err := config.LoadFiles("./auth_config.json")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Config:", config.String("configLoadedMessage"))

	aServer := AuthServer{
		DBInfo: DBInfo{
			DBAddr: config.String("dbAddr"),
			DBPort: config.String("dbPort"),
		},
		Config: ServerConfig{
			dev:                  config.Bool("dev"),
			listenPort:           config.String("serverPort"),
			maxConnections:       config.Int("maxConnections"),
			acceptingConnections: true,
		},
	}

	err = aServer.StartServer()
	if err != nil {
		log.Fatal(err)
	}
}

func HandleConnection(conn net.Conn) {
	fmt.Println("Accepted connection from:", conn.RemoteAddr().String())
}
