package ipdetect

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"strings"
	"time"
)

// dnsQueryTXT sends a DNS TXT query and returns the record content.
func dnsQueryTXT(server, name string) (string, error) {
	id := uint16(time.Now().UnixNano()) //nolint:gosec // DNS query ID, truncation to 16 bits is intentional
	query := buildDNSQuery(id, name)

	// UDP first; fall back to TCP when the response is truncated (TC flag).
	return dnsQueryUDP(server, query, id)
}

// buildDNSQuery builds a DNS query packet (type TXT, class CH).
func buildDNSQuery(id uint16, name string) []byte {
	// Header: ID(2) + Flags(2) + QDCOUNT(2) + ANCOUNT(2) + NSCOUNT(2) + ARCOUNT(2)
	query := make([]byte, 0, 12+len(name)+2+4)
	query = binary.BigEndian.AppendUint16(query, id)
	query = binary.BigEndian.AppendUint16(query, 0x0100) // RD (recursion desired) flag
	query = binary.BigEndian.AppendUint16(query, 1)      // QDCOUNT
	query = binary.BigEndian.AppendUint16(query, 0)      // ANCOUNT
	query = binary.BigEndian.AppendUint16(query, 0)      // NSCOUNT
	query = binary.BigEndian.AppendUint16(query, 0)      // ARCOUNT

	// Encode the domain name.
	for label := range strings.SplitSeq(name, ".") {
		query = append(query, byte(len(label))) //nolint:gosec // label from constant domain name, always <= 63
		query = append(query, label...)
	}

	// Root terminator + query type TXT(16) and class CH(3).
	query = append(query, 0, 0x00, 0x10, 0x00, 0x03)

	return query
}

// dnsQueryUDP sends the DNS query over UDP.
func dnsQueryUDP(server string, query []byte, id uint16) (string, error) {
	conn, err := net.DialTimeout("udp", net.JoinHostPort(server, "53"), detectTimeout)
	if err != nil {
		return "", err
	}
	defer func() {
		_ = conn.Close()
	}()
	_ = conn.SetDeadline(time.Now().Add(detectTimeout))

	if _, err := conn.Write(query); err != nil {
		return "", err
	}

	buf := make([]byte, 4096)
	n, err := conn.Read(buf)
	if err != nil {
		return "", err
	}

	// Retry over TCP when the response is truncated (TC flag).
	if n >= 4 && binary.BigEndian.Uint16(buf[2:4])&0x0200 != 0 {
		return dnsQueryTCP(server, query, id)
	}

	return parseDNSResponse(buf[:n], id)
}

// dnsQueryTCP sends the DNS query over TCP (used when the UDP response
// is truncated).
func dnsQueryTCP(server string, query []byte, id uint16) (string, error) {
	conn, err := net.DialTimeout("tcp", net.JoinHostPort(server, "53"), detectTimeout)
	if err != nil {
		return "", err
	}
	defer func() {
		_ = conn.Close()
	}()
	_ = conn.SetDeadline(time.Now().Add(detectTimeout))

	// TCP messages carry a 2-byte length prefix.
	msg := make([]byte, 0, len(query)+2)
	msg = binary.BigEndian.AppendUint16(msg, uint16(len(query))) //nolint:gosec // query is a ~40-byte built query, no overflow
	msg = append(msg, query...)
	if _, err := conn.Write(msg); err != nil {
		return "", err
	}

	// Read the 2-byte response length prefix first.
	header := make([]byte, 2)
	if _, err := io.ReadFull(conn, header); err != nil {
		return "", err
	}

	buf := make([]byte, binary.BigEndian.Uint16(header))
	if _, err := io.ReadFull(conn, buf); err != nil {
		return "", err
	}

	return parseDNSResponse(buf, id)
}

// parseDNSResponse parses a DNS response and returns the TXT record
// content.
func parseDNSResponse(resp []byte, id uint16) (string, error) {
	if len(resp) < 12 {
		return "", errors.New("DNS response too short")
	}

	if binary.BigEndian.Uint16(resp[0:2]) != id {
		return "", errors.New("DNS response ID mismatch")
	}

	flags := binary.BigEndian.Uint16(resp[2:4])
	if flags&0x8000 == 0 {
		return "", errors.New("not a DNS response")
	}
	if flags&0x000F != 0 {
		return "", fmt.Errorf("DNS response error: rcode=%d", flags&0x000F)
	}

	// Skip the question section.
	offset := skipDNSName(resp, 12)
	if offset < 0 || offset+4 > len(resp) {
		return "", errors.New("invalid DNS question section")
	}
	offset += 4 // skip Type and Class

	// Parse the answer section.
	anCount := int(binary.BigEndian.Uint16(resp[6:8]))
	var txts []string
	for range anCount {
		offset = skipDNSName(resp, offset)
		if offset < 0 || offset+10 > len(resp) {
			return "", errors.New("invalid DNS answer header")
		}

		recordType := binary.BigEndian.Uint16(resp[offset : offset+2])
		rdLength := int(binary.BigEndian.Uint16(resp[offset+8 : offset+10]))
		offset += 10
		if offset+rdLength > len(resp) {
			return "", errors.New("invalid DNS answer data")
		}

		if recordType == 16 { // TXT
			for i := 0; i < rdLength; {
				length := int(resp[offset+i])
				i++
				if i+length > rdLength {
					return "", errors.New("invalid TXT record data")
				}
				txts = append(txts, string(resp[offset+i:offset+i+length]))
				i += length
			}
		}

		offset += rdLength
	}

	if len(txts) == 0 {
		return "", errors.New("no TXT record found")
	}

	return strings.Join(txts, ""), nil
}

// skipDNSName skips a DNS name, supporting compression pointers.
func skipDNSName(msg []byte, offset int) int {
	for offset < len(msg) {
		length := int(msg[offset])
		if length == 0 {
			return offset + 1
		}
		if length&0xC0 == 0xC0 {
			// Compression pointer occupies 2 bytes.
			return offset + 2
		}
		if length > 63 {
			return -1
		}
		offset += 1 + length
	}
	return -1
}
