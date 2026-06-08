package main

import (
	"encoding/binary"
	"fmt"
	"strings"
)

type DNSHeader struct {
	ID      uint16
	QR      bool
	Opcode  uint8
	AA      bool
	TC      bool
	RD      bool
	RA      bool
	RCODE   uint8
	QDCount uint16
	ANCount uint16
	NSCount uint16
	ARCount uint16
}

type DNSQuestion struct {
	Name  string
	Type  uint16
	Class uint16
}

type DNSResourceRecord struct {
	Name  string
	Type  uint16
	Class uint16
	TTL   uint32
	RData []byte
}

type DNSMessage struct {
	Header     DNSHeader
	Questions  []DNSQuestion
	Answers    []DNSResourceRecord
	Authority  []DNSResourceRecord
	Additional []DNSResourceRecord
}

func ParseDNSMessage(buf []byte) (DNSMessage, error) {
	header, err := parseHeader(buf)
	if err != nil {
		return DNSMessage{}, err
	}

	offset := 12 // header is always 12 bytes

	questions := make([]DNSQuestion, 0, header.QDCount)
	for i := 0; i < int(header.QDCount); i++ {
		q, nextOffset, err := parseQuestion(buf, offset)
		if err != nil {
			return DNSMessage{}, err
		}

		questions = append(questions, q)
		offset = nextOffset
	}

	answers := make([]DNSResourceRecord, 0, header.ANCount)
	for i := 0; i < int(header.ANCount); i++ {
		rr, nextOffset, err := parseResourceRecord(buf, offset)
		if err != nil {
			return DNSMessage{}, err
		}

		answers = append(answers, rr)
		offset = nextOffset
	}

	return DNSMessage{
		Header:    header,
		Questions: questions,
		Answers:   answers,
	}, nil
}

func parseHeader(buf []byte) (DNSHeader, error) {
	if len(buf) < 12 {
		return DNSHeader{}, fmt.Errorf("buffer too short for DNS header")
	}

	id := binary.BigEndian.Uint16(buf[0:2])
	flags := binary.BigEndian.Uint16(buf[2:4])

	qr := (flags >> 15) & 0x1
	opcode := (flags >> 11) & 0xF
	aa := (flags >> 10) & 0x1
	tc := (flags >> 9) & 0x1
	rd := (flags >> 8) & 0x1
	ra := (flags >> 7) & 0x1
	rcode := flags & 0xF

	return DNSHeader{
		ID:      id,
		QR:      qr == 1,
		Opcode:  uint8(opcode),
		AA:      aa == 1,
		TC:      tc == 1,
		RD:      rd == 1,
		RA:      ra == 1,
		RCODE:   uint8(rcode),
		QDCount: binary.BigEndian.Uint16(buf[4:6]),
		ANCount: binary.BigEndian.Uint16(buf[6:8]),
		NSCount: binary.BigEndian.Uint16(buf[8:10]),
		ARCount: binary.BigEndian.Uint16(buf[10:12]),
	}, nil
}

func parseName(buf []byte, offset int) (string, int, error) {
	var labels []string
	visited := offset
	jumped := false

	for {
		if offset >= len(buf) {
			return "", 0, fmt.Errorf("offset out of bounds")
		}

		b := buf[offset]

		if b == 0x00 {
			// end of name
			if !jumped {
				visited = offset + 1
			}
			break

		} else if b&0xC0 == 0xC0 {
			// pointer - need 2 bytes
			if offset+1 >= len(buf) {
				return "", 0, fmt.Errorf("pointer out of bounds")
			}

			if !jumped {
				visited = offset + 2 // resume here after name is done
			}

			pointer := uint16(b&0x3F)<<8 | uint16(buf[offset+1])

			offset = int(pointer)
			jumped = true
		} else {
			length := int(b)

			offset++

			if offset+length > len(buf) {
				return "", 0, fmt.Errorf("label out of bounds")
			}

			labels = append(labels, string(buf[offset:offset+length]))

			offset += length
		}
	}

	return strings.Join(labels, "."), visited, nil
}

func parseQuestion(buf []byte, offset int) (DNSQuestion, int, error) {
	name, offset, err := parseName(buf, offset)
	if err != nil {
		return DNSQuestion{}, 0, err
	}

	if offset+4 > len(buf) {
		return DNSQuestion{}, 0, fmt.Errorf("buffer too short for question fields")
	}

	qtype := binary.BigEndian.Uint16(buf[offset : offset+2])
	class := binary.BigEndian.Uint16(buf[offset+2 : offset+4])

	return DNSQuestion{
		Name:  name,
		Type:  qtype,
		Class: class,
	}, offset + 4, nil
}

func parseResourceRecord(buf []byte, offset int) (DNSResourceRecord, int, error) {
	name, offset, err := parseName(buf, offset)
	if err != nil {
		return DNSResourceRecord{}, 0, err
	}

	// TYPE(2) + CLASS(2) + TTL(4) + RDLENGTH(2)
	if offset+10 > len(buf) {
		return DNSResourceRecord{}, 0, fmt.Errorf("buffer too short for RR header")
	}

	rrType := binary.BigEndian.Uint16(buf[offset : offset+2])
	rrClass := binary.BigEndian.Uint16(buf[offset+2 : offset+4])
	ttl := binary.BigEndian.Uint32(buf[offset+4 : offset+8])
	rdLength := binary.BigEndian.Uint16(buf[offset+8 : offset+10])

	offset += 10

	if offset+int(rdLength) > len(buf) {
		return DNSResourceRecord{}, 0, fmt.Errorf("buffer too short for RDATA")
	}

	rData := make([]byte, rdLength)
	copy(rData, buf[offset:offset+int(rdLength)])

	return DNSResourceRecord{
		Name:  name,
		Type:  rrType,
		Class: rrClass,
		TTL:   ttl,
		RData: rData,
	}, offset + int(rdLength), nil
}

func patchResponseID(response []byte, id uint16) {
	if len(response) < 2 {
		return
	}
	response[0] = byte(id >> 8)
	response[1] = byte(id)
}
