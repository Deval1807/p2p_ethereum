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
	// load the env file
	err := godotenv.Load()
	if err != nil {
		fmt.Println("Error loading .env file:", err)
		return
	}
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

	// enodeURL := "enode://79ef83056b3b19d76d3a333e7acf0317ddd636b835c3dd176e62c470419eb2e8025afc8e9e5cf3ae24704502cf6ce2c0546a12b1d518c23c92c4e15925ae7b63@121.138.64.204:30303"
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

			// newBlockHashes := eth.NewBlockHashesPacket{
			// 	{
			// 		Number: 0,
			// 		Hash:   opts.GenesisHash,
			// 	},
			// }

			if err := c.requestNewBlocks(); err != nil {
				return err
			}

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

				insert, err := db.Query("INSERT INTO p2p.p2pmessages (enode, msgcode) VALUES (?, ?);", c.node.URLv4(), msg.Code)
				if err != nil {
					panic(err.Error())
				}
				defer insert.Close()

				switch msg.Code {
				case eth.StatusMsg:
					var status eth.StatusPacket
					if err := msg.Decode(&status); err != nil {
						return err
					}
					// Handle status message
					fmt.Println("Received status message:", status)

				case eth.NewBlockHashesMsg:
					var nhPacket eth.NewBlockHashesPacket
					if err := msg.Decode(&nhPacket); err != nil {
						return err
					}
					// Handle NewBlockHashes message
					fmt.Println("Received NewBlockHashes message:", nhPacket)

				case eth.GetBlockBodiesMsg:
					var gbPacket eth.GetBlockBodiesPacket
					if err := msg.Decode(&gbPacket); err != nil {
						return err
					}
					// Handle GetBlockBodies message
					fmt.Println("Received GetBlockBodies message:", gbPacket)

				case eth.TransactionsMsg:
					var txPacket eth.TransactionsPacket
					if err := msg.Decode(&txPacket); err != nil {
						return err
					}
					// Handle Transactions message
					fmt.Println("Received Transactions message:", txPacket)

				// Add cases for other message types as needed

				default:
					return fmt.Errorf("unknown message type: %d", msg.Code)
				}

			}
		},
	}
}

func PrintMessageType(packet eth.Packet) {
	fmt.Println("Message Type:", packet.Name())
}

func (c *conn) requestNewBlocks() error {
	// Send a message to request new block hashes
	msg := eth.NewBlockHashesPacket{}
	if err := p2p.Send(c.rw, eth.NewBlockHashesMsg, &msg); err != nil {
		return err
	}
	hashes, numbers := msg.Unpack()
	fmt.Println("Requested new blocks and the msg is:  ", msg, hashes, numbers)
	return nil
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
