package logconfig

import (
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"

	rotatelogsbytime "github.com/patch-mirrors/file-rotatelogs"
	"toolkit/log"

	rotators "gopkg.in/natefinch/lumberjack.v2"
)

func init() {
	Register("zap", func(c *Config) (log.FieldLogger, error) {
		writers, err := makeWriters(c)
		if err != nil {
			return nil, err
		}
		return log.NewZapLogger(c.Formatter, c.EnableFileLine, c.Level, writers...), nil
	})
}

type Config struct {
	Driver         string
	Stdout         bool
	Formatter      string
	Level          string
	EnableFileLine bool `mapstructure:"enable_file_line"`
	Path           string
	//MaxAge 日志保存时间 默认15天 单位天
	MaxAge int

	//单位为m 默认200m 分割
	MaxFileSize int
	//文件保存的个数 默认200
	MaxBackup int
	Default   bool
}

type Configs map[string]Config

var (
	instances = map[string]log.FieldLogger{}
	drivers   = map[string]factory{}
	mu        sync.Mutex
)

type factory func(c *Config) (log.FieldLogger, error)

func Register(name string, fn func(c *Config) (log.FieldLogger, error)) {
	mu.Lock()
	defer mu.Unlock()
	drivers[name] = fn
}

func getFactory(driver string) (factory, error) {
	if driver == "" {
		driver = "zap"
	}
	if f, ok := drivers[driver]; ok {
		return f, nil
	}
	return nil, fmt.Errorf("%s driver not found", driver)
}

func Get(name string) (log.FieldLogger, error) {
	res, ok := instances[name]
	if !ok {
		return nil, fmt.Errorf("%s logger not found", name)
	}
	return res, nil
}

func (config *Configs) Init() (err error) {
	if config == nil {
		return nil
	}
	instances = make(map[string]log.FieldLogger, 1)
	for subSection, item := range *config {
		factory, err := getFactory(item.Driver)
		if err != nil {
			return err
		}
		logger, err := factory(&item)
		if err != nil {
			return err
		}

		instances[subSection] = logger
		if item.Default {
			log.Logger.Infof("log [%s] set as default log", subSection)
			log.Logger = logger
		}
		log.Logger.Infof("log [%s] load success", subSection)
	}

	return nil
}

// 热更新
type HotUpdate interface {
	OnChange(c *Config) error
}

func (config Configs) Change() (err error) {
	for subSection, item := range config {
		logger, ok := instances[subSection]
		if !ok {
			fmt.Errorf("log [%s] not found\n", subSection)
			return
		}

		if imp, ok := logger.(HotUpdate); ok {
			err := imp.OnChange(&item)
			if err != nil {
				fmt.Errorf("OnChange [%s:%s] error: %s\n", subSection, item.Driver, err.Error())
			}
		}

	}
	return nil
}

// 关闭
type ServerClose interface {
	OnClose()
}

func (config Configs) Close() {
	for subSection, _ := range config {
		if logger, ok := instances[subSection]; ok {
			if imp, ok := logger.(ServerClose); ok {
				imp.OnClose()
			}
		}
	}
}

// 写入对象
func makeWriters(cfg *Config) (ws []io.Writer, err error) {
	ws = make([]io.Writer, 0)

	path := cfg.Path
	if cfg.MaxAge <= 0 {
		cfg.MaxAge = 15
	}
	if cfg.MaxBackup <= 0 {
		cfg.MaxBackup = 200
	}
	if cfg.MaxFileSize <= 0 {
		cfg.MaxFileSize = 200
	}
	if path != "" {
		var logf io.Writer
		if strings.Index(cfg.Path, "%Y%m%d") >= 0 {
			logf, err = rotatelogsbytime.New(
				cfg.Path,
				rotatelogsbytime.WithRotationCount(uint(cfg.MaxBackup)),
				rotatelogsbytime.WithRotationTime(time.Hour),
			)
			if err != nil {
				return nil, err
			}
		} else {
			logf = &rotators.Logger{
				Filename:   cfg.Path,
				MaxSize:    cfg.MaxFileSize,
				MaxAge:     cfg.MaxAge,
				MaxBackups: cfg.MaxBackup,
				LocalTime:  true,
			}
		}
		ws = append(ws, logf)
	}
	if cfg.Stdout {
		ws = append(ws, os.Stdout)
	}
	return ws, nil
}
