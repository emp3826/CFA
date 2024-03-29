package vmess

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Dreamacro/clash/websocket"
)

type websocketConn struct {
	conn       *websocket.Conn
	reader     io.Reader
	remoteAddr net.Addr

	rMux sync.Mutex
	wMux sync.Mutex
}

type WebsocketConfig struct {
	Host                string
	Port                string
	Path                string
	Headers             http.Header
	TLS                 bool
	TLSConfig           *tls.Config
}

// Read implements net.Conn.Read()
func (wsc *websocketConn) Read(b []byte) (int, error) {
	wsc.rMux.Lock()
	defer wsc.rMux.Unlock()
	for {
		reader, err := wsc.getReader()
		if err != nil {
			return 0, err
		}

		nBytes, err := reader.Read(b)
		if err == io.EOF {
			wsc.reader = nil
			continue
		}
		return nBytes, err
	}
}

// Write implements io.Writer.
func (wsc *websocketConn) Write(b []byte) (int, error) {
	wsc.wMux.Lock()
	defer wsc.wMux.Unlock()
	if err := wsc.conn.WriteMessage(websocket.BinaryMessage, b); err != nil {
		return 0, err
	}
	return len(b), nil
}

func (wsc *websocketConn) Close() error {
	var errors []string
	if err := wsc.conn.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""), time.Now().Add(time.Second*5)); err != nil {
		errors = append(errors, err.Error())
	}
	if err := wsc.conn.Close(); err != nil {
		errors = append(errors, err.Error())
	}
	if len(errors) > 0 {
		return fmt.Errorf("failed to close connection: %s", strings.Join(errors, ","))
	}
	return nil
}

func (wsc *websocketConn) getReader() (io.Reader, error) {
	if wsc.reader != nil {
		return wsc.reader, nil
	}

	_, reader, err := wsc.conn.NextReader()
	if err != nil {
		return nil, err
	}
	wsc.reader = reader
	return reader, nil
}

func (wsc *websocketConn) LocalAddr() net.Addr {
	return wsc.conn.LocalAddr()
}

func (wsc *websocketConn) RemoteAddr() net.Addr {
	return wsc.remoteAddr
}

func (wsc *websocketConn) SetDeadline(t time.Time) error {
	if err := wsc.SetReadDeadline(t); err != nil {
		return err
	}
	return wsc.SetWriteDeadline(t)
}

func (wsc *websocketConn) SetReadDeadline(t time.Time) error {
	return wsc.conn.SetReadDeadline(t)
}

func (wsc *websocketConn) SetWriteDeadline(t time.Time) error {
	return wsc.conn.SetWriteDeadline(t)
}

func streamWebsocketConn(conn net.Conn, c *WebsocketConfig, earlyData *bytes.Buffer) (net.Conn, error) {
	dialer := &websocket.Dialer{
		NetDial: func(network, addr string) (net.Conn, error) {
			return conn, nil
		},
		ReadBufferSize:   4 * 1024,
		WriteBufferSize:  4 * 1024,
		HandshakeTimeout: time.Second * 8,
	}

	scheme := "ws"

	uri := url.URL{
		Scheme: scheme,
		Host:   net.JoinHostPort(c.Host, c.Port),
		Path:   c.Path,
	}

	headers := http.Header{}
	if c.Headers != nil {
		for k := range c.Headers {
			headers.Add(k, c.Headers.Get(k))
		}
	}

	wsConn, resp, err := dialer.Dial(uri.String(), headers)
	if err != nil {
		reason := err.Error()
		if resp != nil {
			reason = resp.Status
		}
		return nil, fmt.Errorf("dial %s error: %s", uri.Host, reason)
	}

	return &websocketConn{
		conn:       wsConn,
		remoteAddr: conn.RemoteAddr(),
	}, nil
}

func StreamWebsocketConn(conn net.Conn, c *WebsocketConfig) (net.Conn, error) {
	if u, err := url.Parse(c.Path); err == nil {
		if q := u.Query(); q.Get("ed") != "" {
			if _, err := strconv.Atoi(q.Get("ed")); err == nil {
				q.Del("ed")
				u.RawQuery = q.Encode()
				c.Path = u.String()
			}
		}
	}

	return streamWebsocketConn(conn, c, nil)
}
