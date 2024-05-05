Steps to run the project

1. Create a new go project `go mod init p2p-query`
2. Copy files above
3. Install dependencies `go mod tidy`
4. Start using `go run .`

This programme will create a very minimalistic p2p node and will try to connect with a particular network (eth mainnet or polygon mainnet). Post establishing connection, it can send/receive messages as and when require. I will have to write the code to handle each message type and act accordingly. 

Note: It will take some time (few mins or maybe hours) to connect to a good peer. Please keep the code running or restarting if it isn't connecting with good peers. Once it does, make sure to note it's enode so that you can hardcode it in the p2p config and try to directly connect to that peer the next time. 