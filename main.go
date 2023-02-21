package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"os"
	"strings"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const terminationCharacter = '\x0a'
const messageBufferSize = 128

var inputFile string
var serverAddress string
var verbosity bool

func init() {
	flag.BoolVar(&verbosity, "v", false, "enable verbose logs")
	flag.StringVar(&inputFile, "file", "", "gcode file that you want to send")
	flag.StringVar(&serverAddress, "address", "127.0.0.1", "IP address or domain for the machine runing Carbide Motion")
}

func initLogger() {
	cfg := zap.NewDevelopmentConfig()
	if !verbosity {
		cfg.Level = zap.NewAtomicLevelAt(zapcore.InfoLevel)
	} else {
		cfg.Level = zap.NewAtomicLevelAt(zapcore.DebugLevel)
	}
	cfg.EncoderConfig = zap.NewProductionEncoderConfig()
	logger, err := cfg.Build()
	if err != nil {
		panic(err)
	}
	zap.ReplaceGlobals(logger)
}

func main() {
	flag.Parse()
	initLogger()
	// Validate input address
	addr, err := net.ResolveTCPAddr("tcp", serverAddress+":6280")
	if err != nil {
		flag.PrintDefaults()
		zap.L().Fatal("Could not resolve input address", zap.String("address", serverAddress))
	}
	// Validate input file
	fileInfo, err := os.Stat(inputFile)
	if err != nil {
		flag.PrintDefaults()
		zap.L().Fatal("Could not find input file", zap.String("file", inputFile))
	}
	input, err := os.Open(inputFile)
	if err != nil {
		flag.PrintDefaults()
		zap.L().Fatal("Could not open input file", zap.String("file", inputFile))
	}
	defer input.Close()
	// Setup server connection
	zap.L().Info("sending gcode file", zap.String("file", inputFile), zap.String("address", serverAddress))
	zap.L().Debug("connecting", zap.String("address", addr.String()))
	conn, err := net.DialTCP("tcp", nil, addr)
	if err != nil {
		zap.L().Error("failed to connect to server", zap.String("address", addr.String()))
		return
	}
	defer conn.Close()
	r := bufio.NewReader(conn)
	w := bufio.NewWriter(conn)
	zap.L().Debug("connected")
	// Ensure that server is ready to receive
	state, err := getState(r)
	if err != nil {
		return
	}
	zap.L().Debug("received state", zap.String("state", state))
	if state != "init" {
		zap.L().Error("cannot start outside of init state", zap.String("state", state))
		return
	}
	// Write header
	header := fmt.Sprintf("GCODE: %s:%d\n", inputFile, fileInfo.Size())
	zap.L().Debug("sending header", zap.String("header", header))
	if _, err := w.Write([]byte(header)); err != nil {
		zap.L().Error("failed sending header", zap.Error(err))
		return
	}
	// Write GCode
	zap.L().Debug("sending gcode", zap.Int64("size", fileInfo.Size()))
	n, err := io.Copy(w, input)
	if err != nil {
		zap.L().Error("failed sending file over connection", zap.Error(err), zap.Int64("size", fileInfo.Size()))
		return
	}
	zap.L().Debug("sent gcode", zap.Int64("size", n))
	// Sent termination signal
	if err := w.WriteByte(terminationCharacter); err != nil {
		zap.L().Error("failed sending termination signal", zap.Error(err))
		return
	}
	// Flush connection
	zap.L().Debug("flushing")
	if err := w.Flush(); err != nil {
		zap.L().Error("failed flushing connection", zap.Error(err))
		return
	}
	// Wait for ACK
	if msg, err := readMessage(r); err != nil {
		return
	} else if msg != "GCODE_ACK" {
		zap.L().Error("did not receive ack", zap.String("message", msg))
		return
	}
	zap.L().Info("done")
}

func readMessage(r io.Reader) (string, error) {
	buffer := make([]byte, messageBufferSize)
	outputBuffer := make([]byte, 0, messageBufferSize)
	n, err := r.Read(buffer)
	if err != nil {
		zap.L().Error("failed to read message", zap.Error(err))
		return "", err
	}
	for i := 0; i < n; i++ {
		if buffer[i] == terminationCharacter {
			zap.L().Debug("found termination character", zap.Int("index", i))
			break
		}
		outputBuffer = append(outputBuffer, buffer[i])
	}
	if len(outputBuffer) >= messageBufferSize {
		zap.L().Error("failed to read message", zap.Error(err))
		return "", errors.New("oversized message")
	}
	return string(outputBuffer), nil
}

var errInvalidStatusMessage = errors.New("invalid status message")

func getState(r io.Reader) (string, error) {
	statusLine, err := readMessage(r)
	if err != nil {
		return "", err
	}
	// Get state
	tokens := strings.Split(statusLine, " ")
	if len(tokens) != 2 {
		zap.L().Error("unexpected number of tokens", zap.String("message", statusLine))
		return "", errInvalidStatusMessage
	}
	if strings.ToUpper(tokens[0]) != "STATE:" {
		zap.L().Error("unexpected message key", zap.String("message", statusLine), zap.String("key", tokens[0]))
		return "", errInvalidStatusMessage
	}
	return strings.ToLower(strings.TrimSpace(tokens[1])), nil
}
