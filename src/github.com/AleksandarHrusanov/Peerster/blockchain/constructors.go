package blockchain

import "github.com/AleksandarHrusanov/Peerster/core"

func CreateBlockPublish(fname string, numBytes int64, metahash []byte) *core.BlockPublish {
	txPub := &core.TxPublish{Name: fname, Size: numBytes, MetafileHash: metahash}
	blockPub := &core.BlockPublish{Transaction: *txPub}
	return blockPub
}

func CreateTLCMessage(gossiper *core.Gossiper, txBlock core.BlockPublish) *core.TLCMessage {
	gossiper.MongeringIDLock.Lock()
	gossiper.CurrentMongeringID++
	gossiper.TlcIDs[gossiper.CurrentMongeringID] = true
	tlc := &core.TLCMessage{Origin: gossiper.Name, ID: gossiper.CurrentMongeringID, Confirmed: -1}
	gossiper.MongeringIDLock.Unlock()

	return tlc
}
