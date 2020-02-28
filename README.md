# Peerster
_Peerster_ is a peer-to-peer application written in Go as part of the course CS-438: Decentralized Systems Engineering at EPFL. _Peerster_ implements some messaging protocols via UDP, allowing for peers to distribute text messages, to send private messages, to share, download and search files, naming with blockchain.

## Gossiping and Rumor Mongering
Gossip protocols are used for robust information exchange in dynamic network topologies where nodes can join/leave, might experience connectivity issues, etc. Such protocols are one of the simplest, fastest, and most reliable algorithms for propagating messages in a way to ensure the nodes in the network each (eventually) receive a copy of every message any of the other nodes has sent.
To achieve that, Peerster uses a rumormongering protocol:
* Every new message received by a peer is resent to a random known peer in the network
* A vector-clock is used to deal with the best-effort nature of UDP, to maintain information about previously seen messages, and to request unseen messages from peers who have seen them before 
* Anti-entropy mechanism is used to periodically issue status messages which trigger exchange of unseen messages between two peers

## Routing
A _destination-sequenced distance vector_ routing scheme is used to enable nodes to send unicast, point-to-point messages to each other. Each node maintains a table of key-value pairs where the key is a destination node and the value is a _next hop_ to reach the desired destination. <br>
**NOTE:** A node knows the address of another node either because they knew them at startup, or because they have previously received a message from them. <br>
In addition, there is an option for a node to periodically send _route rumors_ to announce themselves and to enable other nodes to add them to their routing tables.

## File Sharing
UDP works reliably only with short datagrams, so when a node wants to share a file with other nodes in the network, the file is sent in chunks. Peerster performs the following steps:
* **File Indexing** - Peerster first scans each file, divides it into chunks of 8KB, and computes the SHA-256 hash code of the contents of each chunk.
* **Metafile** - Peerster builds a metafile which contains all of the SHA-256 hashes for the file concatenated with each other

## File Downloading
File downloading is implemented via a one-chunk-at-a-time request/response protocol. At this stage, it is assumed that the requesting node has the _metahash_ of the desired file's metafile. The requesting node sends a request for the metafile's metahash and after receiving it, reads the file chunks' metahashes in order and requests them. On receiving the last chunk, the whole file is reconstructed.

## File Search
Here Peerster is enabled to search files by keywords, using an _expanding-ring flooding scheme_. The searching node simply sends a search request with desired keywords and a given budget. A receiving node searches locally for files matching any of the keywords, decreases the budget of the search request, and redistributes it further to as many of it's neighbors as the remaining budget. If a node has a match, it sends a search reply with information about the chunks of the file it has locally. Peerster supports partial matches - e.g., if node A has the first chunk of a two-chunk file and node B has the second chunk, a subsequent download (once all chunks have been found) would request the two chunks from the respective nodes.

## Naming with a Blockchain
Peerster builds a blockchain of shared files to ensure a globally agreed on name-to-metahash mapping in order to protect against adversarial peers. Whenever a node wants to share a file, it sends a publish block with the file name, file size, and metafile hash and waits for a majority confirmation. <br>
**NOTE:** At this point, the number of peers in the network is fixed and known (e.g. it is assumed that no peers join or leave/drop)

# TODOs
While working on the project I have noted down some future TODOs (e.g. optimizing implementation, fixing issues, additional functionality):
## Code cleaning and simplifications
* Restructure long methods
* Simplify some of the state handling and data structures - e.g. downloading state, file information storage, file searching
* Remove finished downloading states
## Functionality addition/changes
* Add usage of RWMutex in place of regular Mutex where appropriate
* Add proper logging functionality
* Add some error checks - e.g. indexing non-existing files
* Use some receiver methods where appropriate
* Spread TLC messages like regular rumors
* Double check off-by-one errors due to indexing

# Remarks
A small part of the initial codebase has been taken from an anonymous student (allowed by the professor), due to better design and lack of time to restrucutre my original code for the respective functionality. Since then, the code in question has been adapted, changed, and optimized as needed (with some TODOs to further improve/restructure it).
