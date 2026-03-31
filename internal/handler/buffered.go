package handler

import (
	"bufio"
	"io"
	"net"
)

// bufferedConn wraps a net.Conn with a bufio.Reader, preserving any buffered data
type bufferedConn struct {
	net.Conn
	reader *bufio.Reader
}

// Read reads data from the connection, first consuming any buffered data
func (bc *bufferedConn) Read(p []byte) (int, error) {
	return bc.reader.Read(p)
}

// newBufferedConn creates a new buffered connection from a bufio.Reader
func newBufferedConn(conn net.Conn, reader *bufio.Reader) net.Conn {
	return &bufferedConn{
		Conn:   conn,
		reader: reader,
	}
}

// readHTTPResponse reads the HTTP response from a proxy connection
// and returns a connection that includes any buffered data
func readHTTPResponse(conn net.Conn) (net.Conn, *bufio.Reader, error) {
	reader := bufio.NewReader(conn)
	return newBufferedConn(conn, reader), reader, nil
}

// drainAndReturnConn reads remaining headers and returns a connection
// that preserves any buffered data
func drainAndReturnConn(conn net.Conn, reader *bufio.Reader) net.Conn {
	// Check if there's buffered data
	if reader.Buffered() > 0 {
		return newBufferedConn(conn, reader)
	}
	return conn
}

// Ensure bufferedConn implements net.Conn
var _ net.Conn = (*bufferedConn)(nil)

// Optional: implement CloseRead and CloseWrite if the underlying connection supports them
func (bc *bufferedConn) CloseRead() error {
	if cr, ok := bc.Conn.(interface{ CloseRead() error }); ok {
		return cr.CloseRead()
	}
	return nil
}

func (bc *bufferedConn) CloseWrite() error {
	if cw, ok := bc.Conn.(interface{ CloseWrite() error }); ok {
		return cw.CloseWrite()
	}
	return nil
}

// PeekReader wraps a bufio.Reader to peek at data
type PeekReader struct {
	*bufio.Reader
	conn net.Conn
}

// NewPeekReader creates a new PeekReader
func NewPeekReader(conn net.Conn) *PeekReader {
	return &PeekReader{
		Reader: bufio.NewReader(conn),
		conn:   conn,
	}
}

// Conn returns the underlying connection
func (pr *PeekReader) Conn() net.Conn {
	return pr.conn
}

// BufferedConn returns a connection that includes any buffered data
func (pr *PeekReader) BufferedConn() net.Conn {
	return newBufferedConn(pr.conn, pr.Reader)
}

// ReadLine reads a line from the reader
func (pr *PeekReader) ReadLine() (string, error) {
	line, err := pr.Reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	// Remove trailing \r\n or \n
	if len(line) > 0 && line[len(line)-1] == '\n' {
		line = line[:len(line)-1]
	}
	if len(line) > 0 && line[len(line)-1] == '\r' {
		line = line[:len(line)-1]
	}
	return line, nil
}

// SkipHeaders skips HTTP headers until empty line
func (pr *PeekReader) SkipHeaders() error {
	for {
		line, err := pr.Reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}
		if line == "\r\n" || line == "\n" {
			return nil
		}
	}
}
