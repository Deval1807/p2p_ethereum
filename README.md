# Steps to run the project

## Clone the existing repo

## For the Backend Server:
1. Install dependencies `go mod tidy`
2. Start using `go run .`

## For the Frontend:
1. cd into the Frontend folder
2. run `npm i`
3. after the server is started, start frontend by `npm run dev`

### Working:
This programme will create a very minimalistic p2p node and will try to connect with a particular network (eth mainnet or polygon mainnet). Post establishing connection, it can send/receive messages as and when require. 

For simplicity, the current implementation will only connect to 1 particular peer, rather than finding new peer each time.
It will do a status exchange with the connected peer and get the Hash of the Latest Block from the same.

We will get other block information like block number and timestamp from the hash by a 3rd party RPC call (we are using Alchemy).

This data will be sent to frontend and rendered there.