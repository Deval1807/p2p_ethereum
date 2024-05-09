package main

// necessary imports
import (
	"bytes"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/eth/protocols/eth"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/p2p/nat"
	_ "github.com/go-sql-driver/mysql"
	"github.com/joho/godotenv"
)

var latestBlock int
var latestBlockHash common.Hash

func main() {
	// load the env file
	err := godotenv.Load()
	if err != nil {
		fmt.Println("Error loading .env file:", err)
		return
	}

	// connect database
	// DB_URL := os.Getenv("DB_URL")
	// db, err := sql.Open("mysql", DB_URL)
	// if err != nil {
	// 	fmt.Println("error validating sql.Open arguments")
	// 	panic(err.Error())
	// }
	// defer db.Close()
	// err = db.Ping()
	// if err != nil {
	// 	fmt.Println("error verifying the connection with db.Ping")
	// 	panic(err.Error())
	// }
	// fmt.Println("Successful Connection to Database")

	// get the network information
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

	// Generate Private Key
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

	// Static connection to a particular enode
	enodeURL := "enode://46948128c35a1d9b2d05067f70882873f517075dd7709e78f19601244a3741c2d9be34169f67764b42019dbc6ce24dbaa4210ec4f57225df30f04b10f30f59a8@167.179.75.28:30303"
	node, err := enode.ParseV4(enodeURL)
	if err != nil {
		// Handle error
		fmt.Println("Errorrrr :", err)
		return
	}

	// Create the p2p config
	config := p2p.Config{
		PrivateKey: privateKey,
		// BootstrapNodes: n.enodes,
		StaticNodes: []*enode.Node{node},
		MaxPeers:    1,                         // Can be increased later if we want to connect to more nodes
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

	go func() {
		// Register HTTP handlers
		http.HandleFunc("/latest-block-number", getLatestBlockNumberHandler)

		// Start HTTP server
		fmt.Println("Starting HTTP server on port 3001...")
		if err := http.ListenAndServe(":3001", nil); err != nil {
			fmt.Printf("Failed to start HTTP server: %s\n", err)
		}
	}()

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

			// for i := 0; i < 3; i++ {
			// fmt.Println("less gooo", i)
			err := c.statusExchange(&status)
			if err != nil {
				return err
			}

			// time.Sleep(5 * time.Second)

			// }

			fmt.Println("Done status exchange", "enode", c.node.URLv4())

			/*
				// newBlockHashes := eth.NewBlockHashesPacket{
				// 	{
				// 		Number: 0,
				// 		Hash:   opts.GenesisHash,
				// 	},
				// }

				// err1 := c.newBlockExchange(&newBlockHashes)
				// if err1 != nil {
				// 	return err1
				// }

				// fmt.Println("Done new block exchange")

				// Handle all the of the messages here.
				// for {
				// msg, err := rw.ReadMsg()
				// if err != nil {
				// 	return err
				// }
				// _ = msg
				// Handle each message type here and do whatever required (track, log, store, etc.)
				// fmt.Println("msg: ", msg)
				// // Print message type
				// fmt.Println("msg.Code:", msg.Code)

				// insert, err := db.Query("INSERT INTO p2p.p2pmessages (enode, msgcode) VALUES (?, ?);", c.node.URLv4(), msg.Code)
				// if err != nil {
				// 	panic(err.Error())
				// }
				// defer insert.Close()

				// switch msg.Code {
				// case eth.StatusMsg:
				// 	var status eth.StatusPacket
				// 	if err := msg.Decode(&status); err != nil {
				// 		return err
				// 	}
				// 	// Handle status message
				// 	fmt.Println("Received status message:", status)

				// case eth.NewBlockHashesMsg:
				// 	var nhPacket eth.NewBlockHashesPacket
				// 	if err := msg.Decode(&nhPacket); err != nil {
				// 		return err
				// 	}
				// 	// Handle NewBlockHashes message
				// 	fmt.Println("Received NewBlockHashes message:", nhPacket)

				// case eth.GetBlockBodiesMsg:
				// 	var gbPacket eth.GetBlockBodiesPacket
				// 	if err := msg.Decode(&gbPacket); err != nil {
				// 		return err
				// 	}
				// 	// Handle GetBlockBodies message
				// 	fmt.Println("Received GetBlockBodies message:", gbPacket)

				// case eth.TransactionsMsg:
				// 	var txPacket eth.TransactionsPacket
				// 	if err := msg.Decode(&txPacket); err != nil {
				// 		return err
				// 	}
				// 	// Handle Transactions message
				// 	fmt.Println("Received Transactions message:", txPacket)

				// // Add cases for other message types as needed

				// default:
				// 	return fmt.Errorf("unknown message type: %d", msg.Code)
				// }

				// }
			*/
			return nil
		},
	}
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

	timeout := time.NewTimer(1 * time.Second)
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
	fmt.Println("Hash of new block : ", status.Head)

	latestBlockHash = status.Head
	getBlockNumberByHash(latestBlockHash)

	return nil
}

func getBlockNumberByHash(hash common.Hash) {
	RPC_URL := os.Getenv("ALCHEMY_URL")
	body := map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  "eth_getBlockByHash",
		"params": []interface{}{
			hash,
			true,
		},
	}

	// Encode the request body as JSON
	requestBody, err := json.Marshal(body)
	if err != nil {
		fmt.Println("Error encoding request body:", err)
		return
	}

	// Create a new HTTP POST request
	req, err := http.NewRequest("POST", RPC_URL, bytes.NewBuffer(requestBody))
	if err != nil {
		fmt.Println("Error creating request:", err)
		return
	}

	// Set the Content-Type header to specify JSON data
	req.Header.Set("Content-Type", "application/json")

	// Create an HTTP client
	client := &http.Client{}

	// Send the request
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Error sending request:", err)
		return
	}
	defer resp.Body.Close()

	// Check the response status code
	fmt.Println("Response Status:", resp.Status)

	// Decode the response body
	var responseData map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&responseData)
	if err != nil {
		fmt.Println("Error decoding response body:", err)
		return
	}

	// Extract the "number" field from the "result" object
	result := responseData["result"].(map[string]interface{})
	number := result["number"].(string)

	// Print the number
	// fmt.Println("Block Number:", number) // this will be a hexadecimal string
	// fmt.Printf("Type of: %T\n", number)

	// Parse the hexadecimal string to an integer
	numberInt, err := strconv.ParseInt(number, 0, 64)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	latestBlock = int(numberInt)

	fmt.Println("Latest Block Number:", numberInt)
	// os.Exit(1)
}

func enableCors(w *http.ResponseWriter) {
	(*w).Header().Set("Access-Control-Allow-Origin", "*")
}

func getLatestBlockNumberHandler(w http.ResponseWriter, r *http.Request) {
	// enabling CORS
	enableCors(&w)

	// Encode the latest block number as JSON and send it in the response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"latestBlockNumber": latestBlock,
		"latestBlockHash":   latestBlockHash,
	})

}

// Try for direct NewBlockHash msg request
/*
func (c *conn) newBlockExchange(packet *eth.NewBlockHashesPacket) error {
	errc := make(chan error, 2)

	go func() {
		errc <- p2p.Send(c.rw, eth.NewBlockHashesMsg, &packet)
	}()

	go func() {
		errc <- c.readNewBlock(packet)
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

func (c *conn) readNewBlock(packet *eth.NewBlockHashesPacket) error {
	msg, err := c.rw.ReadMsg()
	if err != nil {
		return err
	}

	if msg.Code != eth.NewBlockHashesMsg {
		return errors.New("expected status message code")
	}

	err = msg.Decode(&packet)
	if err != nil {
		return err
	}

	fmt.Println("New Block:  ", packet)
	return nil
}

*/
