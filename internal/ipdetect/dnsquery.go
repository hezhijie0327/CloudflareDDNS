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

// dnsQueryTXT 发送DNS TXT查询并返回记录内容
func dnsQueryTXT(server, name string) (string, error) {
	id := uint16(time.Now().UnixNano()) //nolint:gosec // DNS query ID, truncation to 16 bits is intentional
	query := buildDNSQuery(id, name)

	// 优先UDP查询，响应截断（TC标志）时回退到TCP
	return dnsQueryUDP(server, query, id)
}

// buildDNSQuery 构建DNS查询包（TXT类型，CH类）
func buildDNSQuery(id uint16, name string) []byte {
	// Header: ID(2) + Flags(2) + QDCOUNT(2) + ANCOUNT(2) + NSCOUNT(2) + ARCOUNT(2)
	query := make([]byte, 0, 12+len(name)+2+4)
	query = binary.BigEndian.AppendUint16(query, id)
	query = binary.BigEndian.AppendUint16(query, 0x0100) // 设置RD（期望递归）标志
	query = binary.BigEndian.AppendUint16(query, 1)      // QDCOUNT
	query = binary.BigEndian.AppendUint16(query, 0)      // ANCOUNT
	query = binary.BigEndian.AppendUint16(query, 0)      // NSCOUNT
	query = binary.BigEndian.AppendUint16(query, 0)      // ARCOUNT

	// 编码域名
	for label := range strings.SplitSeq(name, ".") {
		query = append(query, byte(len(label))) //nolint:gosec // label from constant domain name, always <= 63
		query = append(query, label...)
	}

	// 域名根终止符 + 查询类型TXT(16)和类CH(3)
	query = append(query, 0, 0x00, 0x10, 0x00, 0x03)

	return query
}

// dnsQueryUDP 通过UDP发送DNS查询
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

	// 响应被截断（TC标志）时使用TCP重试
	if n >= 4 && binary.BigEndian.Uint16(buf[2:4])&0x0200 != 0 {
		return dnsQueryTCP(server, query, id)
	}

	return parseDNSResponse(buf[:n], id)
}

// dnsQueryTCP 通过TCP发送DNS查询（用于UDP响应截断的情况）
func dnsQueryTCP(server string, query []byte, id uint16) (string, error) {
	conn, err := net.DialTimeout("tcp", net.JoinHostPort(server, "53"), detectTimeout)
	if err != nil {
		return "", err
	}
	defer func() {
		_ = conn.Close()
	}()
	_ = conn.SetDeadline(time.Now().Add(detectTimeout))

	// TCP查询需要在消息前附加2字节长度前缀
	msg := make([]byte, 0, len(query)+2)
	msg = binary.BigEndian.AppendUint16(msg, uint16(len(query))) //nolint:gosec // query is a ~40-byte built query, no overflow
	msg = append(msg, query...)
	if _, err := conn.Write(msg); err != nil {
		return "", err
	}

	// 先读取2字节响应长度前缀
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

// parseDNSResponse 解析DNS响应，返回TXT记录内容
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

	// 跳过问题段
	offset := skipDNSName(resp, 12)
	if offset < 0 || offset+4 > len(resp) {
		return "", errors.New("invalid DNS question section")
	}
	offset += 4 // 跳过Type和Class

	// 解析答案段
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

// skipDNSName 跳过DNS名称，支持压缩指针
func skipDNSName(msg []byte, offset int) int {
	for offset < len(msg) {
		length := int(msg[offset])
		if length == 0 {
			return offset + 1
		}
		if length&0xC0 == 0xC0 {
			// 压缩指针，固定占用2字节
			return offset + 2
		}
		if length > 63 {
			return -1
		}
		offset += 1 + length
	}
	return -1
}
