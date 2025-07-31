package util

import (
	"errors"
	"fmt"
	"os"
	"sync"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const LOG_BUFFER_SIZE = 1000

var (
	ErrLogNotInitialized      = errors.New("log object is not initialized yet")
	LOG_FOLDER_NAME_WITH_PATH = ".." + string(os.PathSeparator) + "log"
	globalLogLevel            = 3
)

const (
	LOG_LEVEL_ERROR = iota + 1
	LOG_LEVEL_WARN
	LOG_LEVEL_INFO
	LOG_LEVEL_DEBUG
)

type MetricsLogger struct {
	logBuffer         chan LeveledLogger
	handle            *os.File
	wg                *sync.WaitGroup
	loggerInitialized bool
	zapLogger         *zap.Logger
}

type LeveledLogger struct {
	level  int
	logMsg string
}

func (m *MetricsLogger) Init(logFileName string, rewrite bool) error {

	var (
		err             error
		fileWithRelPath string
	)
	m.wg = new(sync.WaitGroup)
	m.logBuffer = make(chan LeveledLogger, LOG_BUFFER_SIZE)

	m.handle = nil
	fileWithRelPath = LOG_FOLDER_NAME_WITH_PATH + string(os.PathSeparator) + logFileName

	if rewrite {
		m.handle, err = os.OpenFile(fileWithRelPath,
			os.O_RDWR|os.O_CREATE|os.O_TRUNC,
			0666)
	} else {
		m.handle, err = os.OpenFile(fileWithRelPath,
			os.O_RDWR|os.O_CREATE|os.O_APPEND,
			0666)
	}
	if err != nil {
		return err
	}

	m.zapLoggerInit()

	m.wg.Add(1)
	go m.logWritter()

	m.loggerInitialized = true
	return nil
}

func (m *MetricsLogger) zapLoggerInit() {

	var writer zapcore.WriteSyncer
	config := zap.NewProductionEncoderConfig()
	config.EncodeTime = zapcore.ISO8601TimeEncoder

	config.EncodeLevel = zapcore.CapitalLevelEncoder //To Print level in Uppercase.
	fileEncoder := zapcore.NewConsoleEncoder(config) //To Print Lines in non json format.

	writer = zapcore.AddSync(m.handle)

	core := zapcore.NewTee(
		zapcore.NewCore(fileEncoder, writer, GlobalLogLevelSetter()),
	)
	m.zapLogger = zap.New(core)
	defer m.zapLogger.Sync()
}

func GlobalLogLevelSetter() zapcore.Level {
	var zaplevel zapcore.Level
	if globalLogLevel == LOG_LEVEL_ERROR {
		zaplevel = zapcore.ErrorLevel
	} else if globalLogLevel == LOG_LEVEL_WARN {
		zaplevel = zapcore.WarnLevel
	} else if globalLogLevel == LOG_LEVEL_INFO {
		zaplevel = zapcore.InfoLevel
	} else if globalLogLevel == LOG_LEVEL_DEBUG {
		zaplevel = zapcore.DebugLevel
	}
	return zaplevel
}

func (m *MetricsLogger) logWritter() {
	for logdata := range m.logBuffer {
		if logdata.level == LOG_LEVEL_ERROR {
			m.zapLogger.Error(logdata.logMsg)
		} else if logdata.level == LOG_LEVEL_WARN {
			m.zapLogger.Warn(logdata.logMsg)
		} else if logdata.level == LOG_LEVEL_INFO {
			m.zapLogger.Info(logdata.logMsg)
		} else if logdata.level == LOG_LEVEL_DEBUG {
			m.zapLogger.Debug(logdata.logMsg)
		}
	}
	m.wg.Done()
}

func (m *MetricsLogger) LogEvent(v ...interface{}) error {
	var msg string
	var level int
	var ok bool

	if len(v) == 1 {
		level = LOG_LEVEL_INFO
		msg = fmt.Sprint(v[0])

	} else if len(v) > 1 {
		level, ok = v[0].(int)
		if ok {
			if level == LOG_LEVEL_ERROR || level == LOG_LEVEL_WARN || level == LOG_LEVEL_INFO || level == LOG_LEVEL_DEBUG {
				msg = fmt.Sprintf("%v", v[1:])
			} else {
				level = LOG_LEVEL_INFO
				msg = fmt.Sprintf("%v", v)
			}
		} else {
			level = LOG_LEVEL_INFO
			msg = fmt.Sprintf("%v", v)
		}
		msg = msg[1 : len(msg)-1]
	}

	lobj := LeveledLogger{level, msg}

	if !m.loggerInitialized {
		return ErrLogNotInitialized
	}
	m.logBuffer <- lobj
	return nil
}

func (m *MetricsLogger) DeInit() {

	if !m.loggerInitialized {
		return
	}
	m.loggerInitialized = false
	close(m.logBuffer)
	m.wg.Wait()

	m.handle.Close()

}

func SetCommonLoggerAttributes(GlobalLogLevel int) {
	globalLogLevel = GlobalLogLevel
}

func SetLoggerPath(logPath string) {
	LOG_FOLDER_NAME_WITH_PATH = logPath
}

func CheckAndCreateLogFolder(FolderNameWithPath string) {
	_, err := os.Stat(FolderNameWithPath)

	if os.IsNotExist(err) {
		err := os.MkdirAll(FolderNameWithPath, 0755)
		if err != nil {
			fmt.Println("Failed to create the log folder and Mkdir err :: ", err)
		}
	}
}
