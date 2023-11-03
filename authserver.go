package main

import (
	"authserver/utils"
	"database/sql"
	"encoding/binary"
	"fmt"
	"log"
	"net"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/gookit/config/v2"
)

type AuthServer struct {
	DBInfo DBInfo
	Config ServerConfig
	Realms []Realm
}

type DBInfo struct {
	db                 *sql.DB
	DBConnectionString string
}

type ServerConfig struct {
	dev                  bool
	listenPort           string
	maxConnections       int
	acceptingConnections bool
}

type Realm struct {
	Id                   int
	Name                 string
	Address              string
	LocalAddress         string
	LocalSubnetMask      string
	Port                 int
	Icon                 int
	Realmflags           int
	Timezone             int
	AllowedSecurityLevel int
	Population           float32
	GamebuildMin         int
	GamebuildMax         int
	Flag                 int
	Realmbuilds          string
}

type CMD_AUTH_LOGON_CHANNEL_CLIENT struct {
	Code              uint8
	Protocol          uint8
	Size              uint16
	GameName          []uint8
	Version           []uint8
	Build             uint16
	Platform          []uint8
	Os                []uint8
	Locale            []uint8
	WorldRegionBias   []uint8
	Ip                []uint8
	AccountNameLength uint8
	AccountName       []uint8
}

func (aServer AuthServer) StartServer() error {

	fmt.Println("Starting auth server..")

	conn, err := net.Listen("tcp4", aServer.Config.listenPort)
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	dbConn, err := sql.Open("mysql", aServer.DBInfo.DBConnectionString)
	if err != nil {
		log.Fatal(err)
	}
	defer dbConn.Close()

	dbConn.SetMaxOpenConns(aServer.Config.maxConnections)
	dbConn.SetMaxIdleConns(10)
	dbConn.SetConnMaxLifetime(time.Minute * 2)

	aServer.DBInfo.db = dbConn
	aServer.Realms = FetchRealms(dbConn)

	fmt.Println("Auth Server waiting for connections!")

	for {
		connection, err := conn.Accept()
		if err != nil {
			log.Fatal(err)
		}

		go HandleConnection(connection, &aServer)

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
			DBConnectionString: config.String("dbConnectionString"),
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

func HandleConnection(conn net.Conn, server *AuthServer) {
	fmt.Println("Incoming connection from:", conn.RemoteAddr())

	buffer := make([]byte, 128)

	for {
		bytes, err := conn.Read(buffer)
		if err != nil {
			log.Fatal(err)
		}
		defer conn.Close()

		fmt.Printf("read %v bytes\n", bytes)

		size := []byte{buffer[2], buffer[3]}
		gameName := []byte{buffer[4], buffer[5], buffer[6], buffer[7]}
		vers := []byte{buffer[8], buffer[9], buffer[10]}
		build := []byte{buffer[11], buffer[12]}
		worldRegion := []byte{buffer[25], buffer[26], buffer[27], buffer[28]}
		ip := []byte{buffer[29], buffer[30], buffer[31], buffer[32]}

		packet := CMD_AUTH_LOGON_CHANNEL_CLIENT{
			Code:              buffer[0],                         //0
			Protocol:          buffer[1],                         //1
			Size:              binary.LittleEndian.Uint16(size),  //2-3
			GameName:          gameName,                          //4-7
			Version:           vers,                              //8-10
			Build:             binary.LittleEndian.Uint16(build), //11-12
			Platform:          buffer[13:16],                     //13-16
			Os:                buffer[17:20],                     //17-20
			Locale:            buffer[21:25],                     //21-24
			WorldRegionBias:   worldRegion,                       //25 - 28
			Ip:                ip,                                //29-32
			AccountNameLength: buffer[33],                        //33
			AccountName:       buffer[34:44],
		}

		fmt.Println("Code:", fmt.Sprint(packet.Code))
		fmt.Println("Protocol:", fmt.Sprint(packet.Protocol))
		fmt.Println("Size:", fmt.Sprint(packet.Size))
		fmt.Println("GameName:", string(packet.GameName))
		fmt.Println("Version:", fmt.Sprint(packet.Version))
		fmt.Println("Build:", fmt.Sprint(packet.Build))
		fmt.Println("Platform:", utils.ReverseString(string(packet.Platform)))
		fmt.Println("OS:", utils.ReverseString(string(packet.Os)))
		fmt.Println("Locale:", utils.ReverseString(string(packet.Locale)))
		fmt.Println("WorldRegionBias:", fmt.Sprint(packet.WorldRegionBias))
		fmt.Println("IP:", fmt.Sprint(packet.Ip))
		fmt.Println("AccountNameLength:", fmt.Sprint(packet.AccountNameLength))
		fmt.Println("AccountName:", string(packet.AccountName))

		rows, err := server.DBInfo.db.Query("SELECT v, s FROM account WHERE username = ?", packet.AccountName)
		if err != nil {
			log.Fatal(err)
		}

		defer rows.Close()

		for rows.Next() {
			v := ""
			s := ""
			err := rows.Scan(&v, &s)
			if err != nil {
				log.Fatal(err)
			}

			fmt.Println("Account found!")
			//Implement SRP6...
		}

		resp := []byte{
			0x00,
			0x00,
			3,
		}

		conn.Write(resp)
		return
	}
}

func FetchRealms(db *sql.DB) []Realm {
	fmt.Println("Fetching realms...")

	var realms []Realm

	rows, err := db.Query("SELECT * FROM realmd.realmlist")
	if err != nil {
		log.Fatal(err)
	}

	defer rows.Close()

	for rows.Next() {
		realm := Realm{}
		err = rows.Scan(&realm.Id, &realm.Name, &realm.Address, &realm.LocalAddress, &realm.LocalSubnetMask, &realm.Port, &realm.Icon, &realm.Realmflags, &realm.Timezone, &realm.AllowedSecurityLevel, &realm.Population, &realm.GamebuildMin, &realm.GamebuildMax, &realm.Flag, &realm.Realmbuilds)
		realms = append(realms, realm)
	}

	for _, r := range realms {
		fmt.Println("Realm Added:", r.Name)
	}

	fmt.Println("Total realms loaded:", len(realms))
	return realms
}
