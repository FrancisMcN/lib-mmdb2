package mmdb

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/FrancisMcN/lib-mmdb2/field"
	"github.com/FrancisMcN/lib-mmdb2/node"
	"github.com/FrancisMcN/lib-mmdb2/trie"
	"log"
	"net"
	"time"
)

type MMDB struct {
	Bst        []byte
	Data       []byte
	metadata   Metadata
	PrefixTree *trie.Trie
}

func (m MMDB) GetMetadata() Metadata {
	return m.metadata
}

func NewMMDB() *MMDB {
	return &MMDB{
		PrefixTree: trie.NewTrie(),
		metadata: Metadata{
			RecordSize: 28,
			IpVersion:  6,
			Languages: []string{
				"en",
			},
			BinaryFormatMajorVersion: 2,
			BinaryFormatMinorVersion: 0,
			BuildEpoch:               time.Now(),
			Description: map[string]string{
				"en": "Generated by MCN Ltd.",
			},
		},
	}
}

func (m *MMDB) Load(b []byte) {

	// Find binary search tree section, data section, metadata section
	dataSeparator := make([]byte, 16)
	dataStart := bytes.Index(b, dataSeparator) + 16
	bstEnd := dataStart - 16

	metaSeparator := []byte{0xAB, 0xCD, 0xEF, 'M', 'a', 'x', 'M', 'i', 'n', 'd', '.', 'c', 'o', 'm'}
	metaStart := bytes.LastIndex(b[dataStart:], metaSeparator) + dataStart + len(metaSeparator)

	log.Println(fmt.Sprintf("bst start: 0, bst end: %d", bstEnd))
	log.Println(fmt.Sprintf("data start: %d, data end: %d", dataStart, metaStart))
	log.Println(fmt.Sprintf("meta start: %d, meta end: %d", metaStart, uint32(len(b))))

	m.metadata = ParseMetadata(b[metaStart:])
	log.Println("parsed metadata")
	//m.bst = ParseBst(b[:bstEnd], &m.metadata)
	m.Bst = b[:bstEnd]
	log.Println("parsed bst")
	m.Data = b[dataStart:metaStart]
	log.Println("loaded data")

	dbJson, err := json.Marshal(m.metadata)
	if err != nil {
		log.Fatalf(err.Error())
	}

	//MarshalIndent
	dbJson, err = json.MarshalIndent(m.metadata, "", "  ")
	if err != nil {
		log.Fatalf(err.Error())
	}
	fmt.Printf("%s\n", string(dbJson))
}

func (m MMDB) Query(ip net.IP) field.Field {

	//ip[len(ip)-1-4] = 0
	//ip[len(ip)-1-5] = 0

	nodeCount := m.metadata.NodeCount
	recordSize := m.metadata.RecordSize
	recordBytes := recordSize / 8
	nodeBytes := recordBytes * 2
	if recordSize%8 > 0 {
		nodeBytes++
	}

	offset := uint32(0)
	nid := uint32(0)
	for i := 0; i < 128 && nid < nodeCount /* && offset+uint32(nodeBytes) < uint32(len(m.Bst)) */; i++ {
		n := node.FromBytes(m.Bst[offset:offset+uint32(nodeBytes)], recordSize)

		//if !isSet(ip, i) {
		//	// Choose the left node
		//	offset = uint32(n[0].Uint64()) * uint32(nodeBytes)
		//	nid = uint32(n[0].Uint64())
		//} else {
		//	// Choose the right node
		//	offset = uint32(n[1].Uint64()) * uint32(nodeBytes)
		//	nid = uint32(n[1].Uint64())
		//}
		if !isSet(ip, i) {
			// Choose the left node
			//fmt.Println(i, nid, offset, len(m.Bst),  isSet(ip, i), n, "next node is left node", n[0])
			offset = uint32(n[0].Uint64()) * uint32(nodeBytes)
			nid = uint32(n[0].Uint64())
		} else {
			// Choose the right node
			//fmt.Println(i, nid, offset, len(m.Bst), isSet(ip, i), n, "next node is right node", n[1])
			offset = uint32(n[1].Uint64()) * uint32(nodeBytes)
			nid = uint32(n[1].Uint64())
		}

	}
	if nid == nodeCount {
		fmt.Println("not found")
		return nil
	}
	if nid > nodeCount {
		fp := field.FieldParserSingleton()
		dataOffset := (offset / uint32(nodeBytes)) - 16 - m.metadata.NodeCount
		fp.SetOffset(dataOffset)
		return fp.Parse(m.Data)
	}

	return nil

}

func (m MMDB) PrintMetadata() {

	//MarshalIndent
	dbJson, err := json.MarshalIndent(m.metadata, "", "  ")
	if err != nil {
		log.Fatalf(err.Error())
	}
	fmt.Printf("%s\n", string(dbJson))
}

func (m MMDB) Bytes() []byte {

	b := m.PrefixTree.Bytes()
	m.metadata.NodeCount = m.PrefixTree.Size
	b = append(b, m.metadata.Bytes()...)
	return b

}

// Determines if the 'bit' in the IP is set
// 'bit' is calculated from the most significant byte first
func isSet(ip net.IP, bit int) bool {
	whichByte := bit / 8
	ipByte := ip[whichByte]
	return ((ipByte >> (7 - (bit % 8))) & 1) > 0
}
