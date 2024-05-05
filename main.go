package main

import (
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/eth/protocols/eth"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/nat"
	_ "github.com/go-sql-driver/mysql"
	"github.com/joho/godotenv"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		fmt.Println("Error loading .env file:", err)
		return
	}
	check := os.Getenv("check")
	fmt.Println("check: ", check)
	DB_URL := os.Getenv("DB_URL")
	db, err := sql.Open("mysql", DB_URL)
	if err != nil {
		fmt.Println("error validating sql.Open arguments")
		panic(err.Error())
	}
	defer db.Close()
	err = db.Ping()
	if err != nil {
		fmt.Println("error verifying the connection with db.Ping")
		panic(err.Error())
	}
	fmt.Println("Successful Connection to Database")
	n := getNetworkInfo("eth_mainnet")
	if n == nil {
		fmt.Println("Invalid network")
		return
	}

	// Get the latest block to send the status message
	block, chainId, err := getLatestBlockAndChainId(n.rpcUrl)
	if err != nil {
		fmt.Println("Error fetching latest block:", err)
		return
	}

	// Gather few things required for starting a new p2p node
	opts := EthProtocolOptions{
		GenesisHash: n.genesisHash,
		NetworkID:   chainId.Uint64(),
		Head:        block,
		ForkID:      n.forkId,
	}

	privateKey, err := crypto.GenerateKey()
	if err != nil {
		fmt.Println("Error generating private key:", err)
		return
	}

	nat, err := nat.Parse("any")
	if err != nil {
		fmt.Println("Error parsing nat:", err)
		return
	}

	// enodeURL := "enode://1aa94845dec67a3005370a3bba63cccb86c238a21c682f692d407c0b92cd810a30dba58d31571e38e919a2f3816d521789f904a6ffad5f517f69a17a58dadf93@5.161.220.126:30318"
	// node, err := enode.ParseV4(enodeURL)
	// if err != nil {
	// 	// Handle error
	// 	fmt.Println("Errorrrr :", err)
	// 	return
	// }
	// Create the p2p config
	config := p2p.Config{
		PrivateKey:     privateKey,
		BootstrapNodes: n.enodes,
		// StaticNodes: []*enode.Node{node},
		MaxPeers:    100,                       // Can be increased later if we want to connect to more nodes
		ListenAddr:  fmt.Sprintf(":%d", 30303), // TCP network listening port
		DiscAddr:    fmt.Sprintf(":%d", 30303), // UDP p2p discovery port
		NAT:         nat,
		DiscoveryV4: true,
		DiscoveryV5: true,
		Protocols: []p2p.Protocol{ // Register all 3 eth protocols
			NewEthProtocol(66, opts, db),
			NewEthProtocol(67, opts, db),
			NewEthProtocol(68, opts, db),
		},
	}

	server := p2p.Server{Config: config}

	fmt.Println("Starting node", "enode", server.Self().URLv4())

	// Start the server
	if err := server.Start(); err != nil {
		fmt.Println("Server error:", err)
	}
	defer server.Stop()

	// Setup signals to gracefully stop the node
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)

	for {
		select {
		case <-signals:
			// This gracefully stops the node
			fmt.Println("Stopping node")
			return
		}
	}
}

// NewEthProctocol creates the new eth protocol. This will handle writing the
// status exchange, message handling, and writing blocks/txs to the database.
func NewEthProtocol(version uint, opts EthProtocolOptions, db *sql.DB) p2p.Protocol {
	return p2p.Protocol{
		Name:    "eth",
		Version: version,
		Length:  17,
		Run: func(p *p2p.Peer, rw p2p.MsgReadWriter) error {
			c := conn{
				node: p.Node(),
				rw:   rw,
			}

			status := eth.StatusPacket{
				ProtocolVersion: uint32(version),
				NetworkID:       opts.NetworkID,
				Genesis:         opts.GenesisHash,
				ForkID:          opts.ForkID,
				Head:            opts.Head.Hash(),
				TD:              opts.Head.Difficulty(),
			}
			err := c.statusExchange(&status)
			if err != nil {
				return err
			}

			fmt.Println("Done status exchange", "enode", c.node.URLv4())

			// Handle all the of the messages here.
			for {
				msg, err := rw.ReadMsg()
				if err != nil {
					return err
				}
				_ = msg
				// Handle each message type here and do whatever required (track, log, store, etc.)
				fmt.Println("msg: ", msg)
				// Print message type
				fmt.Println("msg.Code:", msg.Code)
				fmt.Println("msg.String():", msg.String())
				insert, err := db.Query("INSERT INTO p2p.p2pmessages (enode, msgcode) VALUES (?, ?);", c.node.URLv4(), msg.Code)
				if err != nil {
					panic(err.Error())
				}
				defer insert.Close()
			}
		},
	}
}

func PrintMessageType(packet eth.Packet) {
	fmt.Println("Message Type:", packet.Name())
}

// statusExchange will exchange status message between the nodes. It will return
// an error if the nodes are incompatible.
func (c *conn) statusExchange(packet *eth.StatusPacket) error {
	errc := make(chan error, 2)

	go func() {
		errc <- p2p.Send(c.rw, eth.StatusMsg, &packet)
	}()

	go func() {
		errc <- c.readStatus(packet)
	}()

	timeout := time.NewTimer(5 * time.Second)
	defer timeout.Stop()

	for i := 0; i < 2; i++ {
		select {
		case err := <-errc:
			if err != nil {
				return err
			}
		case <-timeout.C:
			return p2p.DiscReadTimeout
		}
	}

	return nil
}

func (c *conn) readStatus(packet *eth.StatusPacket) error {
	msg, err := c.rw.ReadMsg()
	if err != nil {
		return err
	}

	if msg.Code != eth.StatusMsg {
		return errors.New("expected status message code")
	}

	var status eth.StatusPacket
	err = msg.Decode(&status)
	if err != nil {
		return err
	}

	if status.NetworkID != packet.NetworkID {
		return fmt.Errorf("network ID mismatch: %d (!= %d)", status.NetworkID, packet.NetworkID)
	}

	if status.Genesis != packet.Genesis {
		return fmt.Errorf("genesis mismatch: %d (!= %d)", status.Genesis, packet.Genesis)
	}

	if status.ForkID.Hash != packet.ForkID.Hash {
		return fmt.Errorf("fork ID mismatch: %d (!= %d)", status.ForkID.Hash[:], packet.ForkID.Hash[:])
	}

	fmt.Println("New peer connected", "fork_id", hex.EncodeToString(status.ForkID.Hash[:]), "status", status)
	return nil
}
