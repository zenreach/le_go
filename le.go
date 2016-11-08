// Package le_go provides a Golang client library for logging to
// logentries.com over a TCP connection.
//
// it uses an access token for sending log events.
package le_go

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"net"
	"os"
)

// Logger represents a Logentries logger,
// it holds the open TCP connection, access token, prefix and flags.
//
// all Logger operations are thread safe and blocking,
// log operations can be invoked in a non-blocking way by calling them from
// a goroutine.
type Logger struct {
	conn   net.Conn
	flag   int
	prefix string
	token  string
}

const (
	asciiLineSep = 0x0A // "\n"
	asciiSpace   = 0x20 // " "
)

var unicodeLineSep = []byte{0xE2, 0x80, 0xA8} // "\u2028"

// Connect creates a new Logger instance and opens a TCP connection to
// logentries.com,
// The token can be generated at logentries.com by adding a new log,
// choosing manual configuration and token based TCP connection.
func Connect(token string) (*Logger, error) {
	logger := Logger{
		token: token,
	}

	if err := logger.openConnection(); err != nil {
		return nil, err
	}

	return &logger, nil
}

// Close closes the TCP connection to logentries.com
func (logger *Logger) Close() error {
	if logger.conn != nil {
		err := logger.conn.Close()
		logger.conn = nil
		return err
	}

	return nil
}

// Opens a TCP connection to logentries.com
func (logger *Logger) openConnection() error {
	conn, err := tls.Dial("tcp", "data.logentries.com:443", &tls.Config{})
	if err != nil {
		return err
	}
	logger.conn = conn
	return nil
}

// Closes the TCP connection to logentries.com and opens a new one
func (logger *Logger) reopenConnection() error {
	// Continue even if Close fails
	logger.Close()

	return logger.openConnection()
}

// Fatal is same as Print() but calls to os.Exit(1)
func (logger *Logger) Fatal(v ...interface{}) {
	logger.Output(2, fmt.Sprint(v...))
	os.Exit(1)
}

// Fatalf is same as Printf() but calls to os.Exit(1)
func (logger *Logger) Fatalf(format string, v ...interface{}) {
	logger.Output(2, fmt.Sprintf(format, v...))
	os.Exit(1)
}

// Fatalln is same as Println() but calls to os.Exit(1)
func (logger *Logger) Fatalln(v ...interface{}) {
	logger.Output(2, fmt.Sprintln(v...))
	os.Exit(1)
}

// Flags returns the logger flags
func (logger *Logger) Flags() int {
	return logger.flag
}

// Output does the actual writing to the TCP connection
func (logger *Logger) Output(calldepth int, s string) error {
	_, err := logger.Write([]byte(s))

	return err
}

// Panic is same as Print() but calls to panic
func (logger *Logger) Panic(v ...interface{}) {
	s := fmt.Sprint(v...)
	logger.Output(2, s)
	panic(s)
}

// Panicf is same as Printf() but calls to panic
func (logger *Logger) Panicf(format string, v ...interface{}) {
	s := fmt.Sprintf(format, v...)
	logger.Output(2, s)
	panic(s)
}

// Panicln is same as Println() but calls to panic
func (logger *Logger) Panicln(v ...interface{}) {
	s := fmt.Sprintln(v...)
	logger.Output(2, s)
	panic(s)
}

// Prefix returns the logger prefix
func (logger *Logger) Prefix() string {
	return logger.prefix
}

// Print logs a message
func (logger *Logger) Print(v ...interface{}) {
	logger.Output(2, fmt.Sprint(v...))
}

// Printf logs a formatted message
func (logger *Logger) Printf(format string, v ...interface{}) {
	logger.Output(2, fmt.Sprintf(format, v...))
}

// Println logs a message with a linebreak
func (logger *Logger) Println(v ...interface{}) {
	logger.Output(2, fmt.Sprintln(v...))
}

// SetFlags sets the logger flags
func (logger *Logger) SetFlags(flag int) {
	logger.flag = flag
}

// SetPrefix sets the logger prefix
func (logger *Logger) SetPrefix(prefix string) {
	logger.prefix = prefix
}

// Write writes a bytes array to the Logentries TCP connection,
// it adds the access token and prefix and also replaces
// line breaks with the unicode \u2028 character
func (logger *Logger) Write(p []byte) (int, error) {
	buf := logger.makeBuf(p)

	n, err := logger.conn.Write(buf)
	if err == nil {
		return n, err
	}

	// First write failed.  Try reconnecting and then a second write; if that fails give up.  If
	// we wanted to keep trying we would have to maintain a queue and a separate goroutine.
	if err = logger.reopenConnection(); err != nil {
		return 0, err
	}
	n, err = logger.conn.Write(buf)
	return n, err
}

// bytes.IndexByte exists but not bytes.CountByte
func countByte(s []byte, c byte) int {
	return bytes.Count(s, []byte{c})
}

// makeBuf constructs the logger buffer
func (logger *Logger) makeBuf(p []byte) []byte {
	// Pre-allocate a buffer of the correct size
	capacity := len(logger.token) + 1
	capacity += len(logger.prefix) + 1
	capacity += len(p)                         // nominal payload size (before replacement)
	capacity += countByte(p, asciiLineSep) * 2 // 1-byte "\n"s replaced with 3-byte "\u2028"s
	capacity += 1                              // trailing newline
	buf := make([]byte, 0, capacity)

	// Buffer header
	buf = append(buf, logger.token...)
	buf = append(buf, asciiSpace)
	buf = append(buf, logger.prefix...)
	buf = append(buf, asciiSpace)

	// We need to convert the "\n" runes into unicode "\u2028" line separators.  This is done at
	// the byte level to avoid copying data back and forth from strings.
	for {
		i := bytes.IndexByte(p, asciiLineSep)
		if i < 0 {
			buf = append(buf, p...)
			break
		}

		buf = append(buf, p[:i]...)
		buf = append(buf, unicodeLineSep...)
		p = p[i+1:]
	}

	// Buffer must end with an ascii line separator
	buf = append(buf, asciiLineSep)

	return buf
}
