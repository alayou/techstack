package logger

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	ftpserverlog "github.com/fclairamb/go-log"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	lumberjack "gopkg.in/natefinch/lumberjack.v2"
)

const IsoZonedDateTime = "2006-01-02 15:04:05"

// LogLevel defines log levels.
type LogLevel uint8

// defines our own log levels, just in case we'll change logger in future
const (
	LevelDebug LogLevel = iota
	LevelInfo
	LevelWarn
	LevelError
)

var (
	logger        zerolog.Logger
	consoleLogger zerolog.Logger
	rollingLogger *lumberjack.Logger
)

func init() {
	zerolog.TimeFieldFormat = IsoZonedDateTime
	zerolog.SetGlobalLevel(zerolog.ErrorLevel)
	logger = log.Logger.Output(zerolog.ConsoleWriter{
		Out:        os.Stderr,
		NoColor:    true,
		TimeFormat: IsoZonedDateTime,
	}).With().Caller().CallerWithSkipFrameCount(4).Logger()
}

// GetLogger get the configured logger instance
func GetLogger() *zerolog.Logger {
	return &logger
}

// InitLogger 初始化logger.
func InitLogger(level, logPath string) {
	switch level {
	case "info":
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	case "warn":
		zerolog.SetGlobalLevel(zerolog.WarnLevel)
	case "error":
		zerolog.SetGlobalLevel(zerolog.ErrorLevel)
	case "empty":
		zerolog.SetGlobalLevel(zerolog.NoLevel)
	case "debug":
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	}

	// 设置时间格式
	zerolog.TimeFieldFormat = IsoZonedDateTime

	// 设置日志输出方式
	var (
		consoleWriter io.Writer
		logFile       *os.File
		err           error
	)
	// 设置日志文件
	if logPath != "" {
		logFile, err = initLoggerFile(logPath)
		if err != nil {
			log.Error().Err(err).Msg("初始化日志文件失败")
		}
	}

	consoleWriter = zerolog.ConsoleWriter{
		Out:        os.Stdout,
		NoColor:    true,
		TimeFormat: IsoZonedDateTime,
	}

	// 默认都会往控制台输入日志
	var writer io.Writer
	if logFile != nil {
		writer = zerolog.MultiLevelWriter(consoleWriter, logFile)
	} else {
		writer = consoleWriter
	}
	logger = zerolog.New(writer).With().Timestamp().Logger()
	if strings.ToLower(level) == "debug" {
		logger = logger.With().Caller().CallerWithSkipFrameCount(4).Logger()
	}
}

// InitLoggerPlus configures the logger using the given parameters
func InitLoggerPlus(logFilePath string, logMaxSize int, logMaxBackups int, logMaxAge int, logCompress, logUTCTime bool,
	level zerolog.Level,
) {
	SetLogTime(logUTCTime)
	if isLogFilePathValid(logFilePath) {
		logDir := filepath.Dir(logFilePath)
		if _, err := os.Stat(logDir); errors.Is(err, fs.ErrNotExist) {
			err = os.MkdirAll(logDir, os.ModePerm) //nolint
			if err != nil {
				fmt.Printf("unable to create log dir %q: %v", logDir, err)
			}
		}
		rollingLogger = &lumberjack.Logger{
			Filename:   logFilePath,
			MaxSize:    logMaxSize,
			MaxBackups: logMaxBackups,
			MaxAge:     logMaxAge,
			Compress:   logCompress,
			LocalTime:  !logUTCTime,
		}
		logger = zerolog.New(rollingLogger)
		EnableConsoleLogger(level)
	} else {
		logger = zerolog.New(&logSyncWrapper{
			output: os.Stdout,
		})
		consoleLogger = zerolog.Nop()
	}
	logger = logger.Level(level)
}

// InitStdErrLogger configures the logger to write to stderr
func InitStdErrLogger(level zerolog.Level) {
	logger = zerolog.New(&logSyncWrapper{
		output: os.Stderr,
	}).Level(level)
	consoleLogger = zerolog.Nop()
}

// DisableLogger disable the main logger.
// ConsoleLogger will not be affected
func DisableLogger() {
	logger = zerolog.Nop()
	rollingLogger = nil
}

// EnableConsoleLogger enables the console logger
func EnableConsoleLogger(level zerolog.Level) {
	consoleOutput := zerolog.ConsoleWriter{
		Out:        os.Stdout,
		TimeFormat: IsoZonedDateTime,
	}
	consoleLogger = zerolog.New(consoleOutput).With().Timestamp().Logger().Level(level)
}

// RotateLogFile closes the existing log file and immediately create a new one
func RotateLogFile() error {
	if rollingLogger != nil {
		return rollingLogger.Rotate()
	}
	return errors.New("logging to file is disabled")
}

// SetLogTime sets logging time related setting
func SetLogTime(utc bool) {
	if utc {
		zerolog.TimestampFunc = func() time.Time {
			return time.Now().UTC()
		}
	} else {
		zerolog.TimestampFunc = time.Now
	}
}

// Log logs at the specified level for the specified sender
func Log(level LogLevel, sender string, format string, v ...any) {
	var ev *zerolog.Event
	switch level {
	case LevelDebug:
		ev = logger.Debug()
	case LevelInfo:
		ev = logger.Info()
	case LevelWarn:
		ev = logger.Warn()
	case LevelError:
		ev = logger.Error()
	default:
		ev = logger.Error()
	}
	ev.Timestamp().Str("sender", sender)
	ev.Msg(fmt.Sprintf(format, v...))
}

// Debug logs at debug level for the specified sender
func Debug(sender, format string, v ...any) {
	Log(LevelDebug, sender, format, v...)
}

// Info logs at info level for the specified sender
func Info(sender, format string, v ...any) {
	Log(LevelInfo, sender, format, v...)
}

// Warn logs at warn level for the specified sender
func Warn(sender, format string, v ...any) {
	Log(LevelWarn, sender, format, v...)
}

// Error logs at error level for the specified sender
func Error(sender, format string, v ...any) {
	Log(LevelError, sender, format, v...)
}

// DebugToConsole logs at debug level to stdout
func DebugToConsole(format string, v ...any) {
	consoleLogger.Debug().Msg(fmt.Sprintf(format, v...))
}

// InfoToConsole logs at info level to stdout
func InfoToConsole(format string, v ...any) {
	consoleLogger.Info().Msg(fmt.Sprintf(format, v...))
}

// WarnToConsole logs at info level to stdout
func WarnToConsole(format string, v ...any) {
	consoleLogger.Warn().Msg(fmt.Sprintf(format, v...))
}

// ErrorToConsole logs at error level to stdout
func ErrorToConsole(format string, v ...any) {
	consoleLogger.Error().Msg(fmt.Sprintf(format, v...))
}

// CommandLog logs an SFTP/SCP/SSH command
func CommandLog(command, path, target, user, fileMode, connectionID, protocol string, uid, gid int, atime, mtime,
	sshCommand string, size int64, localAddr, remoteAddr string, elapsed int64) {
	logger.Info().
		Timestamp().
		Str("sender", command).
		Str("local_addr", localAddr).
		Str("remote_addr", remoteAddr).
		Str("username", user).
		Str("file_path", path).
		Str("target_path", target).
		Str("filemode", fileMode).
		Int("uid", uid).
		Int("gid", gid).
		Str("access_time", atime).
		Str("modification_time", mtime).
		Int64("size", size).
		Int64("elapsed", elapsed).
		Str("ssh_command", sshCommand).
		Str("connection_id", connectionID).
		Str("protocol", protocol).
		Send()
}

// ConnectionFailedLog logs failed attempts to initialize a connection.
// A connection can fail for an authentication error or other errors such as
// a client abort or a time out if the login does not happen in two minutes.
// These logs are useful for better integration with Fail2ban and similar tools.
func ConnectionFailedLog(user, ip, loginType, protocol, errorString string) {
	logger.Debug().
		Timestamp().
		Str("sender", "connection_failed").
		Str("client_ip", ip).
		Str("username", user).
		Str("login_type", loginType).
		Str("protocol", protocol).
		Str("error", errorString).
		Send()
}

func isLogFilePathValid(logFilePath string) bool {
	cleanInput := filepath.Clean(logFilePath)
	if cleanInput == "." || cleanInput == ".." {
		return false
	}
	return true
}

// StdLoggerWrapper is a wrapper for standard logger compatibility
type StdLoggerWrapper struct {
	Sender string
}

// Write implements the io.Writer interface. This is useful to set as a writer
// for the standard package log.
func (l *StdLoggerWrapper) Write(p []byte) (n int, err error) {
	n = len(p)
	if n > 0 && p[n-1] == '\n' {
		// Trim CR added by stdlog.
		p = p[0 : n-1]
	}

	Log(LevelError, l.Sender, string(p))
	return
}

// LeveledLogger is a logger that accepts a message string and a variadic number of key-value pairs
type LeveledLogger struct {
	Sender            string
	additionalKeyVals []any
}

func addKeysAndValues(ev *zerolog.Event, keysAndValues ...any) {
	kvLen := len(keysAndValues)
	if kvLen%2 != 0 {
		extra := keysAndValues[kvLen-1]
		keysAndValues = append(keysAndValues[:kvLen-1], "EXTRA_VALUE_AT_END", extra)
	}
	for i := 0; i < len(keysAndValues); i += 2 {
		key, val := keysAndValues[i], keysAndValues[i+1]
		if keyStr, ok := key.(string); ok && keyStr != "timestamp" {
			ev.Str(keyStr, fmt.Sprintf("%v", val))
		}
	}
}

// Error logs at error level for the specified sender
func (l *LeveledLogger) Error(msg string, keysAndValues ...any) {
	ev := logger.Error()
	ev.Timestamp().Str("sender", l.Sender)
	if len(l.additionalKeyVals) > 0 {
		addKeysAndValues(ev, l.additionalKeyVals...)
	}
	addKeysAndValues(ev, keysAndValues...)
	ev.Msg(msg)
}

// Info logs at info level for the specified sender
func (l *LeveledLogger) Info(msg string, keysAndValues ...any) {
	ev := logger.Info()
	ev.Timestamp().Str("sender", l.Sender)
	if len(l.additionalKeyVals) > 0 {
		addKeysAndValues(ev, l.additionalKeyVals...)
	}
	addKeysAndValues(ev, keysAndValues...)
	ev.Msg(msg)
}

// Debug logs at debug level for the specified sender
func (l *LeveledLogger) Debug(msg string, keysAndValues ...any) {
	ev := logger.Debug()
	ev.Timestamp().Str("sender", l.Sender)
	if len(l.additionalKeyVals) > 0 {
		addKeysAndValues(ev, l.additionalKeyVals...)
	}
	addKeysAndValues(ev, keysAndValues...)
	ev.Msg(msg)
}

// Warn logs at warn level for the specified sender
func (l *LeveledLogger) Warn(msg string, keysAndValues ...any) {
	ev := logger.Warn()
	ev.Timestamp().Str("sender", l.Sender)
	if len(l.additionalKeyVals) > 0 {
		addKeysAndValues(ev, l.additionalKeyVals...)
	}
	addKeysAndValues(ev, keysAndValues...)
	ev.Msg(msg)
}

// Panic logs the panic at error level for the specified sender
func (l *LeveledLogger) Panic(msg string, keysAndValues ...any) {
	l.Error(msg, keysAndValues...)
}

// With returns a LeveledLogger with additional context specific keyvals
func (l *LeveledLogger) With(keysAndValues ...any) ftpserverlog.Logger {
	return &LeveledLogger{
		Sender:            l.Sender,
		additionalKeyVals: append(l.additionalKeyVals, keysAndValues...),
	}
}

// initLoggerFile 初始化日志文件
func initLoggerFile(logPath string) (logFile *os.File, err error) {
	var location string
	if !filepath.IsAbs(logPath) {
		location, err = os.Executable()
		if err != nil {
			return nil, err
		}
		logPath = filepath.Join(filepath.Dir(location), logPath)
	}
	//log.Debug().Str("log_path", logPath).Key("Setup Logger")
	return os.OpenFile(logPath, os.O_WRONLY|os.O_CREATE|os.O_SYNC|os.O_APPEND, os.ModePerm) // nolint
}
