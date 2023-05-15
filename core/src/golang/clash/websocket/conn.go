// Copyright 2013 The Gorilla WebSocket Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package websocket

import (
	"bufio"
	"encoding/binary"
	"errors"
	"io"
	"io/ioutil"
	"math/rand"
	"net"
	"strconv"
	"sync"
	"time"
)

const (
	// Frame header byte 0 bits from Section 5.2 of RFC 6455
	finalBit = 1 << 7
	rsv1Bit  = 1 << 6
	rsv2Bit  = 1 << 5
	rsv3Bit  = 1 << 4

	// Frame header byte 1 bits from Section 5.2 of RFC 6455
	maskBit = 1 << 7

	maxFrameHeaderSize         = 2 + 8 + 4 // Fixed header + length + mask
	maxControlFramePayloadSize = 125

	writeWait = time.Second

	defaultReadBufferSize  = 4096
	defaultWriteBufferSize = 4096

	continuationFrame = 0
	noFrame           = -1
)

// Close codes defined in RFC 6455, section 11.7.
const (
	CloseNormalClosure           = 1000
)

// The message types are defined in RFC 6455, section 11.8.
const (
	// TextMessage denotes a text data message. The text message payload is
	// interpreted as UTF-8 encoded text data.
	TextMessage = 1

	// BinaryMessage denotes a binary data message.
	BinaryMessage = 2

	// CloseMessage denotes a close control message. The optional message
	// payload contains a numeric code and text. Use the FormatCloseMessage
	// function to format a close message payload.
	CloseMessage = 8
)

// ErrCloseSent is returned when the application writes a message to the
// connection after sending a close message.
var ErrCloseSent = errors.New("websocket: close sent")

// netError satisfies the net Error interface.
type netError struct {
	msg       string
	temporary bool
	timeout   bool
}

func (e *netError) Error() string   { return e.msg }
func (e *netError) Temporary() bool { return e.temporary }
func (e *netError) Timeout() bool   { return e.timeout }

// CloseError represents a close message.
type CloseError struct {
	// Code is defined in RFC 6455, section 11.7.
	Code int

	// Text is the optional text payload.
	Text string
}

var (
	errWriteTimeout        = &netError{msg: "websocket: write timeout", timeout: true, temporary: true}
	errBadWriteOpCode      = errors.New("websocket: bad write message type")
	errWriteClosed         = errors.New("websocket: write closed")
	errInvalidControlFrame = errors.New("websocket: invalid control frame")
)

func newMaskKey() [4]byte {
	n := rand.Uint32()
	return [4]byte{byte(n), byte(n >> 8), byte(n >> 16), byte(n >> 24)}
}

func hideTempErr(err error) error {
	if e, ok := err.(net.Error); ok && e.Temporary() {
		err = &netError{msg: e.Error(), timeout: e.Timeout()}
	}
	return err
}

func isControl(frameType int) bool {
	return frameType == CloseMessage
}

func isData(frameType int) bool {
	return frameType == TextMessage || frameType == BinaryMessage
}

var validReceivedCloseCodes = map[int]bool{
	// see http://www.iana.org/assignments/websocket/websocket.xhtml#close-code-number

	CloseNormalClosure:           true,
}

func isValidReceivedCloseCode(code int) bool {
	return validReceivedCloseCodes[code] || (code >= 3000 && code <= 4999)
}

// BufferPool represents a pool of buffers. The *sync.Pool type satisfies this
// interface.  The type of the value stored in a pool is not specified.
type BufferPool interface {
	// Get gets a value from the pool or returns nil if the pool is empty.
	Get() interface{}
	// Put adds a value to the pool.
	Put(interface{})
}

// writePoolData is the type added to the write buffer pool. This wrapper is
// used to prevent applications from peeking at and depending on the values
// added to the pool.
type writePoolData struct{ buf []byte }

// The Conn type represents a WebSocket connection.
type Conn struct {
	conn        net.Conn
	isServer    bool
	subprotocol string

	// Write fields
	mu            chan struct{} // used as mutex to protect write to conn
	writeBuf      []byte        // frame is constructed in this buffer.
	writePool     BufferPool
	writeBufSize  int
	writeDeadline time.Time
	writer        io.WriteCloser // the current writer returned to the application
	isWriting     bool           // for best-effort concurrent write detection

	writeErrMu sync.Mutex
	writeErr   error

	// Read fields
	reader  io.ReadCloser // the current reader returned to the application
	readErr error
	br      *bufio.Reader
	// bytes remaining in current frame.
	// set setReadRemaining to safely update this value and prevent overflow
	readRemaining int64
	readFinal     bool  // true the current message has more frames.
	readLength    int64 // Message size.
	readLimit     int64 // Maximum message size.
	readMaskPos   int
	readMaskKey   [4]byte
	handleClose   func(int, string) error
	readErrCount  int
	messageReader *messageReader // the current low-level reader
}

func newConn(conn net.Conn, isServer bool, readBufferSize, writeBufferSize int, writeBufferPool BufferPool, br *bufio.Reader, writeBuf []byte) *Conn {

	if br == nil {
		if readBufferSize == 0 {
			readBufferSize = defaultReadBufferSize
		} else if readBufferSize < maxControlFramePayloadSize {
			// must be large enough for control frame
			readBufferSize = maxControlFramePayloadSize
		}
		br = bufio.NewReaderSize(conn, readBufferSize)
	}

	if writeBufferSize <= 0 {
		writeBufferSize = defaultWriteBufferSize
	}
	writeBufferSize += maxFrameHeaderSize

	if writeBuf == nil && writeBufferPool == nil {
		writeBuf = make([]byte, writeBufferSize)
	}

	mu := make(chan struct{}, 1)
	mu <- struct{}{}
	c := &Conn{
		isServer:               isServer,
		br:                     br,
		conn:                   conn,
		mu:                     mu,
		readFinal:              true,
		writeBuf:               writeBuf,
		writePool:              writeBufferPool,
		writeBufSize:           writeBufferSize,
	}
	c.SetCloseHandler(nil)
	return c
}

// setReadRemaining tracks the number of bytes remaining on the connection. If n
// overflows, an ErrReadLimit is returned.
func (c *Conn) setReadRemaining(n int64) error {
	c.readRemaining = n
	return nil
}

// Subprotocol returns the negotiated protocol for the connection.
func (c *Conn) Subprotocol() string {
	return c.subprotocol
}

// Close closes the underlying network connection without sending or waiting
// for a close message.
func (c *Conn) Close() error {
	return c.conn.Close()
}

// LocalAddr returns the local network address.
func (c *Conn) LocalAddr() net.Addr {
	return c.conn.LocalAddr()
}

// RemoteAddr returns the remote network address.
func (c *Conn) RemoteAddr() net.Addr {
	return c.conn.RemoteAddr()
}

// Write methods

func (c *Conn) writeFatal(err error) error {
	err = hideTempErr(err)
	c.writeErrMu.Lock()
	if c.writeErr == nil {
		c.writeErr = err
	}
	c.writeErrMu.Unlock()
	return err
}

func (c *Conn) read(n int) ([]byte, error) {
	p, err := c.br.Peek(n)
	c.br.Discard(len(p))
	return p, err
}

func (c *Conn) write(frameType int, deadline time.Time, buf0, buf1 []byte) error {
	<-c.mu
	defer func() { c.mu <- struct{}{} }()

	c.writeErrMu.Lock()
	err := c.writeErr
	c.writeErrMu.Unlock()
	if err != nil {
		return err
	}

	c.conn.SetWriteDeadline(deadline)
	if len(buf1) == 0 {
		_, err = c.conn.Write(buf0)
	} else {
		err = c.writeBufs(buf0, buf1)
	}
	if err != nil {
		return c.writeFatal(err)
	}
	if frameType == CloseMessage {
		c.writeFatal(ErrCloseSent)
	}
	return nil
}

func (c *Conn) writeBufs(bufs ...[]byte) error {
	b := net.Buffers(bufs)
	_, err := b.WriteTo(c.conn)
	return err
}

// WriteControl writes a control message with the given deadline. The allowed
// message types are CloseMessage, PingMessage and PongMessage.
func (c *Conn) WriteControl(messageType int, data []byte, deadline time.Time) error {
	if !isControl(messageType) {
		return errBadWriteOpCode
	}
	if len(data) > maxControlFramePayloadSize {
		return errInvalidControlFrame
	}

	b0 := byte(messageType) | finalBit
	b1 := byte(len(data))
	if !c.isServer {
		b1 |= maskBit
	}

	buf := make([]byte, 0, maxFrameHeaderSize+maxControlFramePayloadSize)
	buf = append(buf, b0, b1)

	buf = append(buf, data...)

	d := 1000 * time.Hour
	if !deadline.IsZero() {
		d = deadline.Sub(time.Now())
		if d < 0 {
			return errWriteTimeout
		}
	}

	timer := time.NewTimer(d)
	select {
	case <-c.mu:
		timer.Stop()
	case <-timer.C:
		return errWriteTimeout
	}
	defer func() { c.mu <- struct{}{} }()

	c.writeErrMu.Lock()
	err := c.writeErr
	c.writeErrMu.Unlock()
	if err != nil {
		return err
	}

	c.conn.SetWriteDeadline(deadline)
	_, err = c.conn.Write(buf)
	if err != nil {
		return c.writeFatal(err)
	}
	if messageType == CloseMessage {
		c.writeFatal(ErrCloseSent)
	}
	return err
}

// beginMessage prepares a connection and message writer for a new message.
func (c *Conn) beginMessage(mw *messageWriter, messageType int) error {
	// Close previous writer if not already closed by the application. It's
	// probably better to return an error in this situation, but we cannot
	// change this without breaking existing applications.
	if c.writer != nil {
		c.writer.Close()
		c.writer = nil
	}

	if !isControl(messageType) && !isData(messageType) {
		return errBadWriteOpCode
	}

	c.writeErrMu.Lock()
	err := c.writeErr
	c.writeErrMu.Unlock()
	if err != nil {
		return err
	}

	mw.c = c
	mw.frameType = messageType
	mw.pos = maxFrameHeaderSize

	if c.writeBuf == nil {
		wpd, ok := c.writePool.Get().(writePoolData)
		if ok {
			c.writeBuf = wpd.buf
		} else {
			c.writeBuf = make([]byte, c.writeBufSize)
		}
	}
	return nil
}

// NextWriter returns a writer for the next message to send. The writer's Close
// method flushes the complete message to the network.
//
// There can be at most one open writer on a connection. NextWriter closes the
// previous writer if the application has not already done so.
//
// All message types (TextMessage, BinaryMessage, CloseMessage, PingMessage and
// PongMessage) are supported.
func (c *Conn) NextWriter(messageType int) (io.WriteCloser, error) {
	var mw messageWriter
	if err := c.beginMessage(&mw, messageType); err != nil {
		return nil, err
	}
	c.writer = &mw
	return c.writer, nil
}

type messageWriter struct {
	c         *Conn
	pos       int  // end of data in writeBuf.
	frameType int  // type of the current frame.
	err       error
}

func (w *messageWriter) endMessage(err error) error {
	if w.err != nil {
		return err
	}
	c := w.c
	w.err = err
	c.writer = nil
	if c.writePool != nil {
		c.writePool.Put(writePoolData{buf: c.writeBuf})
		c.writeBuf = nil
	}
	return err
}

// flushFrame writes buffered data and extra as a frame to the network. The
// final argument indicates that this is the last frame in the message.
func (w *messageWriter) flushFrame(final bool, extra []byte) error {
	c := w.c
	length := w.pos - maxFrameHeaderSize + len(extra)

	// Check for invalid control frames.
	if isControl(w.frameType) &&
		(!final || length > maxControlFramePayloadSize) {
		return w.endMessage(errInvalidControlFrame)
	}

	b0 := byte(w.frameType)
	if final {
		b0 |= finalBit
	}

	b1 := byte(0)
	if !c.isServer {
		b1 |= maskBit
	}

	// Assume that the frame starts at beginning of c.writeBuf.
	framePos := 0
	if c.isServer {
		// Adjust up if mask not included in the header.
		framePos = 4
	}

	switch {
	case length >= 65536:
		c.writeBuf[framePos] = b0
		c.writeBuf[framePos+1] = b1 | 127
		binary.BigEndian.PutUint64(c.writeBuf[framePos+2:], uint64(length))
	case length > 125:
		framePos += 6
		c.writeBuf[framePos] = b0
		c.writeBuf[framePos+1] = b1 | 126
		binary.BigEndian.PutUint16(c.writeBuf[framePos+2:], uint16(length))
	default:
		framePos += 8
		c.writeBuf[framePos] = b0
		c.writeBuf[framePos+1] = b1 | byte(length)
	}

	// Write the buffers to the connection with best-effort detection of
	// concurrent writes. See the concurrency section in the package
	// documentation for more info.

	c.isWriting = true

	err := c.write(w.frameType, c.writeDeadline, c.writeBuf[framePos:w.pos], extra)

	c.isWriting = false

	if err != nil {
		return w.endMessage(err)
	}

	if final {
		w.endMessage(errWriteClosed)
		return nil
	}

	// Setup for next frame.
	w.pos = maxFrameHeaderSize
	w.frameType = continuationFrame
	return nil
}

func (w *messageWriter) ncopy(max int) (int, error) {
	n := len(w.c.writeBuf) - w.pos
	if n <= 0 {
		if err := w.flushFrame(false, nil); err != nil {
			return 0, err
		}
		n = len(w.c.writeBuf) - w.pos
	}
	if n > max {
		n = max
	}
	return n, nil
}

func (w *messageWriter) Write(p []byte) (int, error) {
	if w.err != nil {
		return 0, w.err
	}

	if len(p) > 2*len(w.c.writeBuf) && w.c.isServer {
		// Don't buffer large messages.
		err := w.flushFrame(false, p)
		if err != nil {
			return 0, err
		}
		return len(p), nil
	}

	nn := len(p)
	for len(p) > 0 {
		n, err := w.ncopy(len(p))
		if err != nil {
			return 0, err
		}
		copy(w.c.writeBuf[w.pos:], p[:n])
		w.pos += n
		p = p[n:]
	}
	return nn, nil
}

func (w *messageWriter) WriteString(p string) (int, error) {
	if w.err != nil {
		return 0, w.err
	}

	nn := len(p)
	for len(p) > 0 {
		n, err := w.ncopy(len(p))
		if err != nil {
			return 0, err
		}
		copy(w.c.writeBuf[w.pos:], p[:n])
		w.pos += n
		p = p[n:]
	}
	return nn, nil
}

func (w *messageWriter) ReadFrom(r io.Reader) (nn int64, err error) {
	if w.err != nil {
		return 0, w.err
	}
	for {
		if w.pos == len(w.c.writeBuf) {
			err = w.flushFrame(false, nil)
			if err != nil {
				break
			}
		}
		var n int
		n, err = r.Read(w.c.writeBuf[w.pos:])
		w.pos += n
		nn += int64(n)
		if err != nil {
			if err == io.EOF {
				err = nil
			}
			break
		}
	}
	return nn, err
}

func (w *messageWriter) Close() error {
	if w.err != nil {
		return w.err
	}
	return w.flushFrame(true, nil)
}

// WriteMessage is a helper method for getting a writer using NextWriter,
// writing the message and closing the writer.
func (c *Conn) WriteMessage(messageType int, data []byte) error {
	w, err := c.NextWriter(messageType)
	if err != nil {
		return err
	}
	if _, err = w.Write(data); err != nil {
		return err
	}
	return w.Close()
}

// SetWriteDeadline sets the write deadline on the underlying network
// connection. After a write has timed out, the websocket state is corrupt and
// all future writes will return an error. A zero value for t means writes will
// not time out.
func (c *Conn) SetWriteDeadline(t time.Time) error {
	c.writeDeadline = t
	return nil
}

// Read methods

func (c *Conn) advanceFrame() (int, error) {
	// 1. Skip remainder of previous frame.

	if c.readRemaining > 0 {
		if _, err := io.CopyN(ioutil.Discard, c.br, c.readRemaining); err != nil {
			return noFrame, err
		}
	}

	// 2. Read and parse first two bytes of frame header.
	// To aid debugging, collect and report all errors in the first two bytes
	// of the header.

	var errors []string

	p, err := c.read(2)
	if err != nil {
		return noFrame, err
	}

	frameType := int(p[0] & 0xf)
	final := p[0]&finalBit != 0
	rsv1 := p[0]&rsv1Bit != 0
	rsv2 := p[0]&rsv2Bit != 0
	rsv3 := p[0]&rsv3Bit != 0
	mask := p[1]&maskBit != 0
	c.setReadRemaining(int64(p[1] & 0x7f))

	if rsv1 {
		errors = append(errors, "RSV1 set")
	}

	if rsv2 {
		errors = append(errors, "RSV2 set")
	}

	if rsv3 {
		errors = append(errors, "RSV3 set")
	}

	switch frameType {
	case CloseMessage:
		if c.readRemaining > maxControlFramePayloadSize {
			errors = append(errors, "len > 125 for control")
		}
		if !final {
			errors = append(errors, "FIN not set on control")
		}
	case TextMessage, BinaryMessage:
		if !c.readFinal {
			errors = append(errors, "data before FIN")
		}
		c.readFinal = final
	case continuationFrame:
		if c.readFinal {
			errors = append(errors, "continuation after FIN")
		}
		c.readFinal = final
	default:
		errors = append(errors, "bad opcode "+strconv.Itoa(frameType))
	}

	if mask != c.isServer {
		errors = append(errors, "bad MASK")
	}

	// 3. Read and parse frame length as per
	// https://tools.ietf.org/html/rfc6455#section-5.2
	//
	// The length of the "Payload data", in bytes: if 0-125, that is the payload
	// length.
	// - If 126, the following 2 bytes interpreted as a 16-bit unsigned
	// integer are the payload length.
	// - If 127, the following 8 bytes interpreted as
	// a 64-bit unsigned integer (the most significant bit MUST be 0) are the
	// payload length. Multibyte length quantities are expressed in network byte
	// order.

	switch c.readRemaining {
	case 126:
		p, err := c.read(2)
		if err != nil {
			return noFrame, err
		}

		if err := c.setReadRemaining(int64(binary.BigEndian.Uint16(p))); err != nil {
			return noFrame, err
		}
	case 127:
		p, err := c.read(8)
		if err != nil {
			return noFrame, err
		}

		if err := c.setReadRemaining(int64(binary.BigEndian.Uint64(p))); err != nil {
			return noFrame, err
		}
	}

	// 4. Handle frame masking.

	if mask {
		c.readMaskPos = 0
		p, err := c.read(len(c.readMaskKey))
		if err != nil {
			return noFrame, err
		}
		copy(c.readMaskKey[:], p)
	}

	// 5. For text and binary messages, enforce read limit and return.

	if frameType == continuationFrame || frameType == TextMessage || frameType == BinaryMessage {

		c.readLength += c.readRemaining

		return frameType, nil
	}

	// 6. Read control frame payload.
	if c.readRemaining > 0 {
		_, err = c.read(int(c.readRemaining))
		c.setReadRemaining(0)
		if err != nil {
			return noFrame, err
		}
	}

	return frameType, nil
}

// NextReader returns the next data message received from the peer. The
// returned messageType is either TextMessage or BinaryMessage.
//
// There can be at most one open reader on a connection. NextReader discards
// the previous message if the application has not already consumed it.
//
// Applications must break out of the application's read loop when this method
// returns a non-nil error value. Errors returned from this method are
// permanent. Once this method returns a non-nil error, all subsequent calls to
// this method return the same error.
func (c *Conn) NextReader() (messageType int, r io.Reader, err error) {
	// Close previous reader, only relevant for decompression.
	if c.reader != nil {
		c.reader.Close()
		c.reader = nil
	}

	c.messageReader = nil
	c.readLength = 0

	for c.readErr == nil {
		frameType, err := c.advanceFrame()
		if err != nil {
			c.readErr = hideTempErr(err)
			break
		}

		if frameType == TextMessage || frameType == BinaryMessage {
			c.messageReader = &messageReader{c}
			c.reader = c.messageReader
			return frameType, c.reader, nil
		}
	}

	// Applications that do handle the error returned from this method spin in
	// tight loop on connection failure. To help application developers detect
	// this error, panic on repeated reads to the failed connection.
	c.readErrCount++
	if c.readErrCount >= 1000 {
		panic("repeated read on failed websocket connection")
	}

	return noFrame, nil, c.readErr
}

type messageReader struct{ c *Conn }

func (r *messageReader) Read(b []byte) (int, error) {
	c := r.c
	if c.messageReader != r {
		return 0, io.EOF
	}

	for c.readErr == nil {

		if c.readRemaining > 0 {
			if int64(len(b)) > c.readRemaining {
				b = b[:c.readRemaining]
			}
			n, err := c.br.Read(b)
			c.readErr = hideTempErr(err)
			rem := c.readRemaining
			rem -= int64(n)
			c.setReadRemaining(rem)
			return n, c.readErr
		}

		if c.readFinal {
			c.messageReader = nil
			return 0, io.EOF
		}

		frameType, err := c.advanceFrame()
		switch {
		case err != nil:
			c.readErr = hideTempErr(err)
		case frameType == TextMessage || frameType == BinaryMessage:
			c.readErr = errors.New("websocket: internal error, unexpected text or binary in Reader")
		}
	}

	err := c.readErr
	return 0, err
}

func (r *messageReader) Close() error {
	return nil
}

// ReadMessage is a helper method for getting a reader using NextReader and
// reading from that reader to a buffer.
func (c *Conn) ReadMessage() (messageType int, p []byte, err error) {
	var r io.Reader
	messageType, r, err = c.NextReader()
	if err != nil {
		return messageType, nil, err
	}
	p, err = ioutil.ReadAll(r)
	return messageType, p, err
}

// SetReadDeadline sets the read deadline on the underlying network connection.
// After a read has timed out, the websocket connection state is corrupt and
// all future reads will return an error. A zero value for t means reads will
// not time out.
func (c *Conn) SetReadDeadline(t time.Time) error {
	return c.conn.SetReadDeadline(t)
}

// SetReadLimit sets the maximum size in bytes for a message read from the peer. If a
// message exceeds the limit, the connection sends a close message to the peer
// and returns ErrReadLimit to the application.
func (c *Conn) SetReadLimit(limit int64) {
	c.readLimit = limit
}

// CloseHandler returns the current close handler
func (c *Conn) CloseHandler() func(code int, text string) error {
	return c.handleClose
}

// SetCloseHandler sets the handler for close messages received from the peer.
// The code argument to h is the received close code or CloseNoStatusReceived
// if the close message is empty. The default close handler sends a close
// message back to the peer.
//
// The handler function is called from the NextReader, ReadMessage and message
// reader Read methods. The application must read the connection to process
// close messages as described in the section on Control Messages above.
//
// The connection read methods return a CloseError when a close message is
// received. Most applications should handle close messages as part of their
// normal error handling. Applications should only set a close handler when the
// application must perform some action before sending a close message back to
// the peer.
func (c *Conn) SetCloseHandler(h func(code int, text string) error) {
	if h == nil {
		h = func(code int, text string) error {
			message := FormatCloseMessage(code, "")
			c.WriteControl(CloseMessage, message, time.Now().Add(writeWait))
			return nil
		}
	}
	c.handleClose = h
}

// NetConn returns the underlying connection that is wrapped by c.
// Note that writing to or reading from this connection directly will corrupt the
// WebSocket connection.
func (c *Conn) NetConn() net.Conn {
	return c.conn
}

// FormatCloseMessage formats closeCode and text as a WebSocket close message.
// An empty message is returned for code CloseNoStatusReceived.
func FormatCloseMessage(closeCode int, text string) []byte {
	buf := make([]byte, 2+len(text))
	binary.BigEndian.PutUint16(buf, uint16(closeCode))
	copy(buf[2:], text)
	return buf
}
