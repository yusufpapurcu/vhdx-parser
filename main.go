package main

import (
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
)

var VHDX_HEADER_BLOCK_SIZE = 64 * 1024

var (
	REGION_GUID_BAT      = "2DC27766-F623-4200-9D64-115E9BFD4A08"
	REGION_GUID_METADATA = "8B7CA206-4790-4B9A-B8FE-575F050F886E"
)

type ParentLocatorHeader struct {
	// Offset: 0 | Lenght: 16
	LocatorType []byte
	// Offset: 16 | Lenght: 2
	Reserved []byte
	// Offset: 18 | Lenght: 2
	KeyValueCount []byte
}

func ParentLocatorHeaderFromByteArray(buf []byte) ParentLocatorHeader {
	return ParentLocatorHeader{
		LocatorType:   buf[0:16],
		Reserved:      buf[16 : 16+2],
		KeyValueCount: buf[18 : 18+2],
	}
}

func (locator ParentLocatorHeader) String() string {
	return fmt.Sprintf("==========\nParentLocatorHeader\n\nLocatorType: %x\nEntryCount: %v\n", (locator.LocatorType), binary.LittleEndian.Uint16(locator.KeyValueCount))
}

type MetadataHeader struct {
	// Offset: 0 | Lenght: 8
	Signature []byte
	// Offset: 8 | Lenght: 2
	Reserved []byte
	// Offset: 10 | Lenght: 2
	EntryCount []byte
	// Offset: 12 | Lenght: 20
	Reserved2 []byte
}

func MetadataHeaderFromByteArray(buf []byte) MetadataHeader {
	return MetadataHeader{
		Signature:  buf[0:8],
		Reserved:   buf[8 : 8+2],
		EntryCount: buf[10 : 10+2],
		Reserved2:  buf[12 : 12+20],
	}
}

func (metadata MetadataHeader) String() string {
	return fmt.Sprintf("==========\nMetadataHeader\n\nSignature: %s\nEntryCount: %v\n", string(metadata.Signature), binary.LittleEndian.Uint16(metadata.EntryCount))
}

type MetadataEntry struct {
	// Offset: 0 | Lenght: 16
	ItemID []byte
	// Offset: 16 | Lenght: 4
	Offset []byte
	// Offset: 20 | Lenght: 4
	Lenght []byte
	// Offset: 24 | Lenght: 1
	IsUser []byte
	// Offset: 25 | Lenght: 1
	IsVirtualDisk []byte
	// Offset: 26 | Lenght: 1
	IsRequired []byte
	// Offset: 27 | Lenght: 31
	Reserved []byte
}

func MetadataEntryFromByteArray(buf []byte) MetadataEntry {
	return MetadataEntry{
		ItemID:        buf[0:16],
		Offset:        buf[16 : 16+4],
		Lenght:        buf[20 : 20+4],
		IsUser:        buf[24:25],
		IsVirtualDisk: buf[25:26],
		IsRequired:    buf[26:27],
		Reserved:      buf[27:32],
	}
}

func (metadata MetadataEntry) String() string {
	return fmt.Sprintf("==========\nMetadataEntry\n\nItemID: %x\nOffset: %v\nLenght: %v\n", (metadata.ItemID), binary.LittleEndian.Uint32(metadata.Offset), binary.LittleEndian.Uint32(metadata.Lenght))
}

type Metadata struct {
	ParentLocatorString string
	BlockSize           int64
	HasParent           byte
	LeaveBlockAllocated byte
	LogicalSectorSize   int64
	PhysicalSectorSize  int64
	VirtualDiskSize     int64
}

func main() {
	f, _ := os.Open("../example_data/Fixed2_8F0BD8D7-8051-46EA-95EF-7601DE0F2B12.avhdx")
	defer f.Close()

	f.Seek(int64(VHDX_HEADER_BLOCK_SIZE*3+16), io.SeekCurrent)

	buf := make([]byte, 64)
	f.Read(buf)

	var skipsize uint64
	//fmt.Printf("First region guid: %x\n", buf[0:16])
	if fmt.Sprintf("%x", buf[0:16]) == guidToBlob(REGION_GUID_METADATA) {
		skipsize = binary.LittleEndian.Uint64(buf[16 : 16+8])
	} else {
		skipsize = binary.LittleEndian.Uint64(buf[48 : 48+8])
	}
	(f.Seek(int64(skipsize), io.SeekStart))

	buf = make([]byte, 32)
	f.Read(buf)
	a := MetadataHeaderFromByteArray(buf)
	var Entries []MetadataEntry
	for i := 0; uint16(i) < binary.LittleEndian.Uint16(a.EntryCount); i++ {
		buf = make([]byte, 32)
		f.Read(buf)
		Entries = append(Entries, MetadataEntryFromByteArray(buf))
	}

	var metadata = Metadata{}
	for _, entry := range Entries {
		switch fmt.Sprintf("%x", entry.ItemID) {
		case guidToBlob("CAA16737-FA36-4D43-B3B6-33F0AA44E76B"): // File Parameters
			//fmt.Println("File Parameters")
			f.Seek(int64(skipsize)+int64(binary.LittleEndian.Uint32(entry.Offset)), io.SeekStart)
			buf = make([]byte, binary.LittleEndian.Uint32(entry.Lenght))
			f.Read(buf)
			metadata.BlockSize = int64(binary.LittleEndian.Uint32(buf[0:4]))
			metadata.LeaveBlockAllocated = buf[5]
			metadata.HasParent = buf[6]

		case guidToBlob("2FA54224-CD1B-4876-B211-5DBED83BF4B8"): // Virtual Disk Size
			//fmt.Println("Virtual Disk Size")
			f.Seek(int64(skipsize)+int64(binary.LittleEndian.Uint32(entry.Offset)), io.SeekStart)
			buf = make([]byte, binary.LittleEndian.Uint32(entry.Lenght))
			f.Read(buf)
			metadata.VirtualDiskSize = int64(binary.LittleEndian.Uint64(buf[0:8]))

		case guidToBlob("BECA12AB-B2E6-4523-93EF-C309E000C746"): // Virtual Disk ID
			//fmt.Println("Virtual Disk ID")
			f.Seek(int64(skipsize)+int64(binary.LittleEndian.Uint32(entry.Offset)), io.SeekStart)
			buf = make([]byte, binary.LittleEndian.Uint32(entry.Lenght))
			f.Read(buf)

		case guidToBlob("8141BF1D-A96F-4709-BA47-F233A8FAAB5F"): // Logical Sector Size
			//fmt.Println("Logical Sector Size")
			f.Seek(int64(skipsize)+int64(binary.LittleEndian.Uint32(entry.Offset)), io.SeekStart)
			buf = make([]byte, binary.LittleEndian.Uint32(entry.Lenght))
			f.Read(buf)
			metadata.LogicalSectorSize = int64(binary.LittleEndian.Uint32(buf[0:4]))

		case guidToBlob("CDA348C7-445D-4471-9CC9-E9885251C556"): // Physical Sector Size
			//fmt.Println("Physical Sector Size")
			f.Seek(int64(skipsize)+int64(binary.LittleEndian.Uint32(entry.Offset)), io.SeekStart)
			buf = make([]byte, binary.LittleEndian.Uint32(entry.Lenght))
			f.Read(buf)
			metadata.PhysicalSectorSize = int64(binary.LittleEndian.Uint32(buf[0:4]))

		case guidToBlob("A8D35F2D-B30B-454D-ABF7-D3D84834AB0C"): // Parent Locator
			//fmt.Println("Parent Locator")
			f.Seek(int64(skipsize)+int64(binary.LittleEndian.Uint32(entry.Offset)), io.SeekStart)
			bufParent := make([]byte, binary.LittleEndian.Uint32(entry.Lenght))
			f.Read(bufParent)
			locatorHeader := ParentLocatorHeaderFromByteArray(bufParent[0:20])
			result := ""
			for i, key := range bufParent[int(binary.LittleEndian.Uint16(locatorHeader.KeyValueCount))*12+20:] {
				if i%2 == 0 {
					result += string(key)
				}
			}
			metadata.ParentLocatorString = result
		}
	}
	aa, _ := json.Marshal(metadata)
	fmt.Println(string(aa))
}

// 2097168
// 2097152
func guidToBlob(guid string) string {
	buf, _ := hex.DecodeString(strings.ReplaceAll(guid, "-", ""))
	if len(buf) != 16 {
		return ""
	}

	var new = []byte{}
	new = append(new, reverse(buf[0:4])...)
	new = append(new, reverse(buf[4:6])...)
	new = append(new, reverse(buf[6:8])...)
	new = append(new, buf[8:16]...)

	return fmt.Sprintf("%x", new)
}

func reverse(s []byte) []byte {
	for i, j := 0, len(s)-1; i < j; i, j = i+1, j-1 {
		s[i], s[j] = s[j], s[i]
	}

	return s
}
