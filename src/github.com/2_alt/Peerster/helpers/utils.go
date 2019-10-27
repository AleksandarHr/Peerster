package helpers

import (
	"fmt"
	"math/rand"
	"net"
	"strings"
)

// HandleErrorFatal Handle an error
func HandleErrorFatal(err error) {
	if err != nil {
		panic("Error: " + err.Error())
	}
}

// HandleErrorNonFatal Handle an error without stopping the program
func HandleErrorNonFatal(err error) {
	if err != nil {
		fmt.Println(err)
	}
}

// CreateStringKnownPeers Create a string from the map of known peers
func CreateStringKnownPeers(knownPeers []string) string {
	stringPeers := ""
	if len(knownPeers) == 0 {
		return stringPeers
	}
	for _, peerAddress := range knownPeers {
		stringPeers += peerAddress
		stringPeers += ","
	}
	stringPeers = stringPeers[:len(stringPeers)-1]
	return stringPeers
}

// SliceContainsString Check if a slice contains a string
func SliceContainsString(sx []string, s string) bool {
	for _, str := range sx {
		if strings.Compare(str, s) == 0 {
			return true
		}
	}
	return false
}

// MapContainsString Check if a map contains a string in its keys
func MapContainsString(m map[string]uint32, s string) bool {
	for st := range m {
		if strings.Compare(st, s) == 0 {
			return true
		}
	}
	return false
}

// PickRandomInMap Pick a random key in a map
func PickRandomInMap(m map[string]uint32) string {
	var list []string
	for mapElem := range m {
		list = append(list, mapElem)
	}
	return PickRandomInSlice(list)
}

// PickRandomInSlice Pick a random string in a []string
func PickRandomInSlice(sx []string) string {
	if (len(sx)) == 0 {
		return ""
	}
	randomInt := rand.Intn(len(sx))
	return sx[randomInt]
}

// PickRandomInSliceDifferentFrom Pick a random string in a []string different from a given string
func PickRandomInSliceDifferentFrom(sx []string, notThisOne string) string {
	if len(sx) <= 1 {
		return ""
	}
	currentAddr := notThisOne
	for strings.Compare(notThisOne, currentAddr) == 0 {
		currentAddr = PickRandomInSlice(sx)
	}
	return currentAddr
}

// IPAddressIsValid Check if a given ip4 address is valid
func IPAddressIsValid(address string) bool {
	_, err := net.ResolveUDPAddr("udp4", address)
	return err == nil
}

// VerifyRemoveDuplicateAddrInSlice Remove duplicate addresses in a slice of strings.
// It also removes all invalid ip4 addresses from the list
func VerifyRemoveDuplicateAddrInSlice(sx []string) []string {
	newList := make([]string, 0)
	mapElem := make(map[string]bool)
	for _, address := range sx {
		_, present := mapElem[address]
		if !present && IPAddressIsValid(address) {
			mapElem[address] = true
			newList = append(newList, address)
		}
	}
	return newList
}
