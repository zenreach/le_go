package logentries

import (
	"fmt"
	"io"
	"net"
	"os"
	"sync"
	"time"
)

type Logger struct {
	conn   net.Conn
	flag   int
	mu     sync.Mutex
	prefix string
	token  string
	buf    []byte
}

func Connect(token string) (*Logger, error) {
	Logger := Logger{
		token: token,
	}

	if err := Logger.reopenConnection(); err != nil {
		return nil, err
	}

	return &Logger, nil
}

func (logger *Logger) Close() error {
	if logger.conn != nil {
		return logger.conn.Close()
	}

	return nil
}

func (logger *Logger) reopenConnection() error {
	conn, err := net.Dial("tcp", "data.logentries.com:80")
	if err != nil {
		return err
	}

	logger.conn = conn

	return nil
}

func (logger *Logger) isOpenConnection() bool {
	if logger.conn == nil {
		return false
	}

	buf := make([]byte, 1)

	logger.conn.SetReadDeadline(time.Now())

	if _, err := logger.conn.Read(buf); err.(net.Error).Timeout() == true &&
		err != io.EOF {

		logger.conn.SetReadDeadline(time.Time{})

		return true
	} else {
		logger.conn.Close()

		return false
	}
}

func (logger *Logger) ensureOpenConnection() error {
	if !logger.isOpenConnection() {
		if err := logger.reopenConnection(); err != nil {
			return err
		}
	}

	return nil
}

func (logger *Logger) Fatal(v ...interface{}) {
	logger.Output(2, fmt.Sprint(v...))
	os.Exit(1)
}

func (logger *Logger) Fatalf(format string, v ...interface{}) {
	logger.Output(2, fmt.Sprintf(format, v...))
	os.Exit(1)
}

func (logger *Logger) Fatalln(v ...interface{}) {
	logger.Output(2, fmt.Sprintln(v...))
	os.Exit(1)
}

func (logger *Logger) Flags() int {
	return logger.flag
}

func (logger *Logger) Output(calldepth int, s string) error {
	if err := logger.ensureOpenConnection(); err != nil {
		return err
	}

	logger.mu.Lock()
	defer logger.mu.Unlock()

	logger.buf = logger.buf[:0]
	logger.buf = append(logger.buf, (logger.token + " ")...)
	logger.buf = append(logger.buf, (logger.prefix + " ")...)
	logger.buf = append(logger.buf, s...)

	_, err := logger.conn.Write(logger.buf)

	return err
}

func (logger *Logger) Panic(v ...interface{}) {
	logger.Output(2, fmt.Sprint(v...))
	panic("")
}

func (logger *Logger) Panicf(format string, v ...interface{}) {
	logger.Output(2, fmt.Sprintf(format, v...))
	panic("")
}

func (logger *Logger) Panicln(v ...interface{}) {
	logger.Output(2, fmt.Sprintln(v...))
	panic("")
}

func (logger *Logger) Prefix() string {
	return logger.prefix
}

func (logger *Logger) Print(v ...interface{}) {
	logger.Output(2, fmt.Sprint(v...))
}

func (logger *Logger) Printf(format string, v ...interface{}) {
	logger.Output(2, fmt.Sprintf(format, v...))
}

func (logger *Logger) Println(v ...interface{}) {
	logger.Output(2, fmt.Sprintln(v...))
}

func (logger *Logger) SetFlags(flag int) {
	logger.flag = flag
}

func (logger *Logger) SetPrefix(prefix string) {
	logger.prefix = prefix
}
