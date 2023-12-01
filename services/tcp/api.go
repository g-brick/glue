package tcp

import (
	"errors"
	"net"
)

func SetTCPClientConnLimit(limit int) {
	setClientConnectionLimit(limit)
}

// 带回包的请求
func SendTCPRequest(s_peer_addr string, i_peer_port int, req []byte, timeOut uint32) (res []byte, err error) {
	return sendClientRequest(s_peer_addr, i_peer_port, req, timeOut)
}

// 不带回包的请求
func SendTCPRequestNoResponse(s_peer_addr string, i_peer_port int, req []byte, timeOut uint32) (err error) {
	return sendClientRequestNoResponse(s_peer_addr, i_peer_port, req, timeOut)
}

func SendTCPResponse(connection *TcpConnection, res []byte) (err error) {
	_, err = connection.Send(res)
	return
}

func ParseListenAddr(listen_addr string) (listen_ip string, err error) {
	return parseListenAddr(listen_addr)
}

func IsIPv4(ip string) bool {
	return isIPv4(ip)
}

func NewTcpConnection(conn net.Conn) *TcpConnection {
	return newTcpConnection(conn)
}

//ratelimit
func InitRateLimit(strategy string, rate float32, capacity int64) error {
	if rate <= 0.0 || capacity <= 0 {
		return errors.New("ratelimiter input error")
	}
	defaultStrategy = strategy
	defaultCapacity = capacity
	defaultRate = rate
	return nil
}
