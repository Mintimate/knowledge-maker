package logger

import (
	"fmt"
	"log"
	"os"
	"time"

	"knowledge-maker/internal/config"
)

// Logger 统一日志管理器
type Logger struct {
	config   *config.LogConfig
	fileLog  *log.Logger
	logFile  *os.File
	logLevel LogLevel
}

// LogLevel 日志级别
type LogLevel int

const (
	DEBUG LogLevel = iota
	INFO
	WARN
	ERROR
)

var (
	globalLogger *Logger
	levelNames   = map[LogLevel]string{
		DEBUG: "DEBUG",
		INFO:  "INFO",
		WARN:  "WARN",
		ERROR: "ERROR",
	}
)

// Init 初始化全局日志器
func Init(cfg *config.LogConfig) error {
	logger, err := NewLogger(cfg)
	if err != nil {
		return err
	}
	globalLogger = logger
	return nil
}

// NewLogger 创建新的日志器
func NewLogger(cfg *config.LogConfig) (*Logger, error) {
	// 创建日志目录
	if err := os.MkdirAll(cfg.Dir, 0755); err != nil {
		return nil, fmt.Errorf("创建日志目录失败: %v", err)
	}

	// 创建日志文件
	logFileName := fmt.Sprintf("%s/app_%s.log", cfg.Dir, time.Now().Format("2006-01-02"))
	logFile, err := os.OpenFile(logFileName, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return nil, fmt.Errorf("创建日志文件失败: %v", err)
	}

	// 解析日志级别
	level := parseLogLevel(cfg.Level)

	logger := &Logger{
		config:   cfg,
		fileLog:  log.New(logFile, "", log.LstdFlags),
		logFile:  logFile,
		logLevel: level,
	}

	return logger, nil
}

// parseLogLevel 解析日志级别
func parseLogLevel(level string) LogLevel {
	switch level {
	case "debug":
		return DEBUG
	case "info":
		return INFO
	case "warn":
		return WARN
	case "error":
		return ERROR
	default:
		return INFO
	}
}

// Close 关闭日志器
func (l *Logger) Close() error {
	if l.logFile != nil {
		return l.logFile.Close()
	}
	return nil
}

// Debug 调试日志
func (l *Logger) Debug(format string, args ...interface{}) {
	if l.logLevel <= DEBUG {
		l.writeLog(DEBUG, format, args...)
	}
}

// Info 信息日志
func (l *Logger) Info(format string, args ...interface{}) {
	if l.logLevel <= INFO {
		l.writeLog(INFO, format, args...)
	}
}

// Warn 警告日志
func (l *Logger) Warn(format string, args ...interface{}) {
	if l.logLevel <= WARN {
		l.writeLog(WARN, format, args...)
	}
}

// Error 错误日志
func (l *Logger) Error(format string, args ...interface{}) {
	if l.logLevel <= ERROR {
		l.writeLog(ERROR, format, args...)
	}
}

// writeLog 写入日志
func (l *Logger) writeLog(level LogLevel, format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	logMsg := fmt.Sprintf("[%s] %s", levelNames[level], msg)
	
	// 同时输出到控制台和文件
	log.Printf(logMsg)
	if l.fileLog != nil {
		l.fileLog.Printf(logMsg)
	}
}

// 全局日志函数
func Debug(format string, args ...interface{}) {
	if globalLogger != nil {
		globalLogger.Debug(format, args...)
	} else {
		log.Printf("[DEBUG] "+format, args...)
	}
}

func Info(format string, args ...interface{}) {
	if globalLogger != nil {
		globalLogger.Info(format, args...)
	} else {
		log.Printf("[INFO] "+format, args...)
	}
}

func Warn(format string, args ...interface{}) {
	if globalLogger != nil {
		globalLogger.Warn(format, args...)
	} else {
		log.Printf("[WARN] "+format, args...)
	}
}

func Error(format string, args ...interface{}) {
	if globalLogger != nil {
		globalLogger.Error(format, args...)
	} else {
		log.Printf("[ERROR] "+format, args...)
	}
}

// Close 关闭全局日志器
func Close() error {
	if globalLogger != nil {
		return globalLogger.Close()
	}
	return nil
}