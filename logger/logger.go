package logger

import (
	"os"
	"strings"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/buffer"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

const (
	DebugLevel  = "debug"
	InfoLevel   = "info"
	WarnLevel   = "warn"
	ErrorLevel  = "error"
	DPanicLevel = "dpanic"
	PanicLevel  = "panic"
	FatalLevel  = "fatal"
)

var (
	DefaultLogger *Logger
)

// logLevelMap 日志级别对应关系.
var logLevelMap = map[string]zapcore.Level{
	DebugLevel:  zap.DebugLevel,
	InfoLevel:   zap.InfoLevel,
	WarnLevel:   zap.WarnLevel,
	ErrorLevel:  zap.ErrorLevel,
	DPanicLevel: zap.DPanicLevel,
	PanicLevel:  zap.PanicLevel,
	FatalLevel:  zap.FatalLevel,
}

func init() {
	DefaultLogger = NewLogger(OptionCallerSkip(2))
}

// Logger 日志.
type Logger struct {
	log *zap.SugaredLogger
}

// Config 配置Logger的选项.
type Config struct {
	Level         string // Level 日志等级.
	ConsoleOutput string // 控制台输出流：stdout/stderr，默认是stdout.
	Filename      string // Filename 用于存储日志的文件，可以为空，表示日志不写入文件.
	MaxSize       int    // 单个日志文件最大大小，单位M,
	MaxAge        int    // 日志最大保留时长，单位：天
	MaxBackups    int    // 最大备份数据：份数（如果时间超过了仍然会被删除）
	Compress      bool   // 日志是否开启压缩
	CallerSkip    int    // 报错代码产生跳过层级
}

type Option interface {
	apply(*Config)
}

type optionFunc func(*Config)

func (f optionFunc) apply(c *Config) {
	f(c)
}

func OptionCallerSkip(skip int) Option {
	return optionFunc(func(c *Config) {
		c.CallerSkip = skip
	})
}

func OptionLevel(level string) Option {
	return optionFunc(func(c *Config) {
		c.Level = level
	})
}

func OptionConsoleOutput(consl string) Option {
	return optionFunc(func(c *Config) {
		c.ConsoleOutput = consl
	})
}

func OptionFilename(filename string) Option {
	return optionFunc(func(c *Config) {
		c.Filename = filename
	})
}

func OptionMaxSize(maxSize int) Option {
	return optionFunc(func(c *Config) {
		c.MaxSize = maxSize
	})
}

func OptionMaxAge(maxAge int) Option {
	return optionFunc(func(c *Config) {
		c.MaxAge = maxAge
	})
}

func OptionMaxBackups(maxBackups int) Option {
	return optionFunc(func(c *Config) {
		c.MaxBackups = maxBackups
	})
}

func OptionCompress(compress bool) Option {
	return optionFunc(func(c *Config) {
		c.Compress = compress
	})
}

// NewLogger new Logger instance.
func NewLogger(opts ...Option) *Logger {
	conf := &Config{
		Level:      InfoLevel,
		Filename:   "",
		MaxSize:    5,
		MaxAge:     1,
		MaxBackups: 1,
		Compress:   false,
	}

	for _, opt := range opts {
		opt.apply(conf)
	}

	encoderConfig := zapcore.EncoderConfig{
		MessageKey:     "msg",
		LevelKey:       "level",
		TimeKey:        "date",
		CallerKey:      "file",
		NameKey:        "name",
		FunctionKey:    "func",
		StacktraceKey:  "stack",
		EncodeLevel:    zapcore.CapitalLevelEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
		EncodeTime:     zapcore.RFC3339TimeEncoder,
		EncodeDuration: zapcore.SecondsDurationEncoder,
	}

	// 设置日志级别.
	atomicLevel := zap.NewAtomicLevel()
	atomicLevel.SetLevel(logLevelMap[strings.ToLower(conf.Level)])

	cores := make([]zapcore.Core, 0)
	if conf.Filename != "" {
		fileWriter := &lumberjack.Logger{
			Filename:   conf.Filename,
			MaxSize:    conf.MaxSize,
			MaxBackups: conf.MaxBackups,
			MaxAge:     conf.MaxAge,
			Compress:   conf.Compress,
		}

		fileCore := zapcore.NewCore(
			NewJSONEncoder(encoderConfig),                            // 编码器配置.
			zapcore.NewMultiWriteSyncer(zapcore.AddSync(fileWriter)), // 打印到控制台和文件.
			atomicLevel, // 日志级别.
		)
		cores = append(cores, fileCore)
	}

	consoleWriter := os.Stdout
	if conf.ConsoleOutput == "stderr" {
		consoleWriter = os.Stderr
	}

	consoleCore := zapcore.NewCore(
		NewJSONEncoder(encoderConfig),                               // 编码器配置， zap有json,控制台，map object三种编码器
		zapcore.NewMultiWriteSyncer(zapcore.AddSync(consoleWriter)), // 打印到控制台和文件.
		atomicLevel, // 日志级别.
	)
	cores = append(cores, consoleCore)

	core := zapcore.NewTee(cores...)

	// 构造日志.
	return &Logger{
		log: zap.New(core, zap.AddCaller(), zap.AddCallerSkip(conf.CallerSkip), zap.Development(), zap.AddStacktrace(zap.ErrorLevel)).Sugar(),
	}
}

type JSONEncoder struct {
	zapcore.Encoder
}

func NewJSONEncoder(cfg zapcore.EncoderConfig) *JSONEncoder {
	return &JSONEncoder{zapcore.NewJSONEncoder(cfg)}
}
func (j *JSONEncoder) EncodeEntry(ent zapcore.Entry, fields []zapcore.Field) (*buffer.Buffer, error) {
	tsField := zapcore.Field{
		Key:  "ts",
		Type: zapcore.Int64Type,
		// String:time.Now().Format(time.RFC3339),
		Integer: time.Now().UnixMilli(),
	}
	fields = append(fields, tsField)
	return j.Encoder.EncodeEntry(ent, fields)
}

// Named adds a sub-scope to the logger's name. See Logger.Named for details.
func SetNamed(name string) *Logger {
	return DefaultLogger.Named(name)
}

// With adds a variadic number of fields to the logging context. It accepts a
// mix of strongly-typed Field objects and loosely-typed key-value pairs. When
// processing pairs, the first element of the pair is used as the field key
// and the second as the field value.
//
// For example,
//   Logger.With(
//     "hello", "world",
//     "failure", errors.New("oh no"),
//     Stack(),
//     "count", 42,
//     "user", User{Name: "alice"},
//  )
// is the equivalent of
//   unsugared.With(
//     String("hello", "world"),
//     String("failure", "oh no"),
//     Stack(),
//     Int("count", 42),
//     Object("user", User{Name: "alice"}),
//   )
//
// Note that the keys in key-value pairs should be strings. In development,
// passing a non-string key panics. In production, the logger is more
// forgiving: a separate error is logged, but the key-value pair is skipped
// and execution continues. Passing an orphaned key triggers similar behavior:
// panics in development and errors in production.
func With(args ...interface{}) *Logger {
	return DefaultLogger.With(args...)
}

func (l *Logger) Named(name string) *Logger {
	return &Logger{log: l.log.Named(name)}
}

func (l *Logger) With(args ...interface{}) *Logger {
	return &Logger{log: l.log.With(args...)}
}

func (l *Logger) Debug(args ...interface{}) {
	l.log.Debug(args...)
}

func (l *Logger) Debugf(template string, args ...interface{}) {
	l.log.Debugf(template, args...)
}

func (l *Logger) Debugw(msg string, keysAndValues ...interface{}) {
	l.log.Debugw(msg, keysAndValues...)
}

func (l *Logger) Warn(args ...interface{}) {
	l.log.Warn(args...)
}

func (l *Logger) Warnf(template string, args ...interface{}) {
	l.log.Warnf(template, args...)
}

func (l *Logger) Warnw(msg string, keysAndValues ...interface{}) {
	l.log.Warnw(msg, keysAndValues...)
}

func (l *Logger) Info(args ...interface{}) {
	l.log.Info(args...)
}

func (l *Logger) Infof(template string, args ...interface{}) {
	l.log.Infof(template, args...)
}

func (l *Logger) Infow(msg string, keysAndValues ...interface{}) {
	l.log.Infow(msg, keysAndValues...)
}

func (l *Logger) Error(args ...interface{}) {
	l.log.Error(args...)
}

func (l *Logger) Errorf(template string, args ...interface{}) {
	l.log.Errorf(template, args...)
}

func (l *Logger) Errorw(msg string, keysAndValues ...interface{}) {
	l.log.Errorw(msg, keysAndValues...)
}

func (l *Logger) DPanic(args ...interface{}) {
	l.log.DPanic(args...)
}

func (l *Logger) DPanicf(template string, args ...interface{}) {
	l.log.DPanicf(template, args...)
}

func (l *Logger) DPanicw(msg string, keysAndValues ...interface{}) {
	l.log.DPanicw(msg, keysAndValues...)
}

func (l *Logger) Panic(args ...interface{}) {
	l.log.Panic(args...)
}

func (l *Logger) Panicf(template string, args ...interface{}) {
	l.log.Panicf(template, args...)
}

func (l *Logger) Panicw(msg string, keysAndValues ...interface{}) {
	l.log.Panicw(msg, keysAndValues...)
}

func (l *Logger) Fatal(args ...interface{}) {
	l.log.Fatal(args...)
}

func (l *Logger) Fatalf(template string, args ...interface{}) {
	l.log.Fatalf(template, args...)
}

func (l *Logger) Fatalw(msg string, keysAndValues ...interface{}) {
	l.log.Fatalw(msg, keysAndValues...)
}

// Debug uses fmt.Sprint to construct and log a message.
func Debug(args ...interface{}) {
	DefaultLogger.Debug(args...)
}

// Debugf uses fmt.Sprintf to log a templated message.
func Debugf(template string, args ...interface{}) {
	DefaultLogger.Debugf(template, args...)
}

// Debugw logs a message with some additional context. The variadic key-value
// pairs are treated as they are in With.
//
// When debug-level logging is disabled, this is much faster than
//  s.With(keysAndValues).Debug(msg)
func Debugw(msg string, keysAndValues ...interface{}) {
	DefaultLogger.Debugw(msg, keysAndValues...)
}

// Info uses fmt.Sprint to construct and log a message.
func Info(args ...interface{}) {
	DefaultLogger.Info(args...)
}

// Infof uses fmt.Sprintf to log a templated message.
func Infof(template string, args ...interface{}) {
	DefaultLogger.Infof(template, args...)
}

func Infow(msg string, keysAndValues ...interface{}) {
	DefaultLogger.Infow(msg, keysAndValues...)
}

// Warn uses fmt.Sprint to construct and log a message.
func Warn(args ...interface{}) {
	DefaultLogger.Warn(args...)
}

// Warnf uses fmt.Sprintf to log a templated message.
func Warnf(template string, args ...interface{}) {
	DefaultLogger.Warnf(template, args...)
}

func Warnw(msg string, keysAndValues ...interface{}) {
	DefaultLogger.Warnw(msg, keysAndValues...)
}

// Error uses fmt.Sprint to construct and log a message.
func Error(args ...interface{}) {
	DefaultLogger.Error(args...)
}

// Errorf uses fmt.Sprintf to log a templated message.
func Errorf(template string, args ...interface{}) {
	DefaultLogger.Errorf(template, args...)
}

func Errorw(msg string, keysAndValues ...interface{}) {
	DefaultLogger.Errorw(msg, keysAndValues...)
}

// DPanic uses fmt.Sprint to construct and log a message. In development, the
// logger then panics. (See DPanicLevel for details.)
func DPanic(args ...interface{}) {
	DefaultLogger.DPanic(args...)
}

// DPanicf uses fmt.Sprintf to log a templated message. In development, the
// logger then panics. (See DPanicLevel for details.)
func DPanicf(template string, args ...interface{}) {
	DefaultLogger.DPanicf(template, args...)
}

// DPanicw logs a message with some additional context. In development, the
// logger then panics. (See DPanicLevel for details.) The variadic key-value
// pairs are treated as they are in With.
func DPanicw(msg string, keysAndValues ...interface{}) {
	DefaultLogger.DPanicw(msg, keysAndValues...)
}

// Panic uses fmt.Sprint to construct and log a message, then panics.
func Panic(args ...interface{}) {
	DefaultLogger.Panic(args...)
}

// Panicf uses fmt.Sprintf to log a templated message, then panics.
func Panicf(template string, args ...interface{}) {
	DefaultLogger.Panicf(template, args...)
}

// Panicw logs a message with some additional context, then panics. The
// variadic key-value pairs are treated as they are in With.
func Panicw(msg string, keysAndValues ...interface{}) {
	DefaultLogger.Panicw(msg, keysAndValues...)
}

func Fatal(args ...interface{}) {
	DefaultLogger.Fatal(args...)
}

func Fatalf(template string, args ...interface{}) {
	DefaultLogger.Fatalf(template, args...)
}

func Fatalw(msg string, kvs ...interface{}) {
	DefaultLogger.Fatalw(msg, kvs...)
}

// Sync flushes any buffered log entries.
func Sync() error {
	return DefaultLogger.Sync()
}

func (l *Logger) Sync() error {
	return l.log.Sync()
}
