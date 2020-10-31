package config

import (
	"context"
	"errors"
	"fmt"
	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
	"github.com/zhouchang2017/toolkit/log"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
)

// 初始化
type InitCallback interface {
	Init() (err error)
}

// 配置动态更新
type OnChangeCallback interface {
	Change() (err error)
}

// 关闭
type OnCloseCallback interface {
	Close() (err error)
}

type Config struct {
	Name string
}

func Init(ctx context.Context, config interface{}, configName string, envPrefix string) error {
	if config == nil {
		panic("config is nil")
	}

	c := Config{
		Name: configName,
	}

	// 初始化配置文件
	if err := c.initConfig(ctx, config, envPrefix); err != nil {
		return err
	}

	// 监控配置文件变化并热加载程序
	c.watchConfig(config)

	return nil

}

func getInitDir() ([]string, error) {
	var binDir string
	ret := make([]string, 0)

	appPath, err := filepath.Abs(os.Args[0])
	if err != nil {
		return ret, err
	}

	if strings.Contains(appPath, "go-build") {
		binDir, err = os.Getwd() //假如跑 go run ... 当前目录
		if err != nil {
			return ret, err
		}
	} else if strings.Contains(appPath, "go_build") {
		// goland run windows
		binDir, err = os.Getwd() //假如跑 go run ... 当前目录
		if err != nil {
			return ret, err
		}
	} else {

		binDir = filepath.Dir(appPath)
	}

	var configPath string

	//目录顺序
	// 1: ../conf
	// 2: ./conf
	// 3: ./
	configPath = filepath.Join(filepath.Dir(binDir), "conf") // 1. ../conf
	ret = append(ret, configPath)

	configPath = filepath.Join(binDir, "conf") // 2. ./conf
	ret = append(ret, configPath)

	ret = append(ret, binDir) //3. ./

	return ret, nil

}

func (c *Config) initConfig(ctx context.Context, config interface{}, envPrefix string) error {
	if c.Name != "" {
		viper.SetConfigFile(c.Name) // 如果指定了配置文件，则解析指定的配置文件
	} else {
		// 设置默认查找路径

		dirs, err := getInitDir()
		if err != nil {
			fmt.Printf("getInitDir error  %s\n", err.Error())
			viper.AddConfigPath("../conf") // 如果没有指定配置文件，则解析默认的配置文件
			viper.AddConfigPath("./conf")
			viper.AddConfigPath("./")
		} else {
			for _, dir := range dirs {
				viper.AddConfigPath(dir)
				fmt.Printf("config dir %s\n", dir)
			}
		}

		// 设置配置文件名
		viper.SetConfigName("config")
	}
	viper.SetConfigType("yaml") // 设置配置文件格式为YAML
	viper.AutomaticEnv()        // 读取匹配的环境变量
	if envPrefix == "" {
		envPrefix = strings.ToUpper(filepath.Base(os.Args[0])) + "_"
	}
	// log.Logger.Info("environment prefix: " + envPrefix)
	viper.SetEnvPrefix(envPrefix) // 读取环境变量的前缀
	replacer := strings.NewReplacer(".", "_")
	viper.SetEnvKeyReplacer(replacer)
	if err := viper.ReadInConfig(); err != nil { // viper解析配置文件
		return err
	}

	usedConfig := viper.ConfigFileUsed()
	log.Logger.Infof("Using config: %s", usedConfig)

	configDir := path.Dir(usedConfig)
	includes := viper.GetStringSlice("include")
	if len(includes) > 0 {
		log.Logger.Infof("include: %v", includes)
		for _, globpath := range includes {
			var files, _ = filepath.Glob(configDir + "/" + globpath)
			sort.Strings(files)
			for _, includeFile := range files {
				if f, err := os.Open(includeFile); err != nil {
					log.Logger.Errorf("read config:%s error happens: %s", includeFile, err.Error())
					continue
				} else {
					if err := viper.MergeConfig(f); err != nil {
						log.Logger.Error(err.Error())
						return err
					}
					log.Logger.Infof("%s loaded", includeFile)
					//log.Logger.Infof( "after All: %v", viper.AllSettings())
				}
			}
		}
	} else {
		log.Logger.Info("skip include for [include] not found.")
	}

	if err := viper.Unmarshal(config); err != nil {
		return err
	}

	return LoadInitialCallbacks(config)
}

func LoadInitialCallbacks(config interface{}) error {
	value := reflect.ValueOf(config)

	if value.Type().Kind() != reflect.Ptr {
		return errors.New("Pointer type required")
	}

	value = value.Elem()

	for i := 0; i < value.NumField(); i++ {
		v := value.Field(i)

		if v.IsNil() {
			name := value.Type().Field(i).Name
			log.Logger.Debugf("%s config not found, check if [%s] contains valid section [%s].", name, viper.ConfigFileUsed(), strings.ToLower(name))
		}
		f := v.Interface()
		initCB, ok := f.(InitCallback)
		if ok {
			err := initCB.Init()
			if err != nil {
				return err
			}
			continue
		}
	}

	return nil
}

func LoadOnCloseCallbacks(config interface{}) error {
	value := reflect.ValueOf(config)

	if value.Type().Kind() != reflect.Ptr {
		return errors.New("Pointer type required")
	}

	value = value.Elem()

	for i := 0; i < value.NumField(); i++ {
		v := value.Field(i)

		if v.IsNil() {
			name := value.Type().Field(i).Name
			log.Logger.Debugf("%s config not found, check if [%s] contains valid section [%s].", name, viper.ConfigFileUsed(), strings.ToLower(name))
		}
		f := v.Interface()
		initCB, ok := f.(OnCloseCallback)
		if ok {
			err := initCB.Close()
			if err != nil {
				return err
			}
			continue
		}
	}

	return nil
}

// 监控配置文件变化并热加载程序
func (c *Config) watchConfig(config interface{}) {
	viper.WatchConfig()
	viper.OnConfigChange(func(e fsnotify.Event) {
		log.Logger.Infof("config file changed")
		if err := viper.Unmarshal(config); err != nil {
			log.Logger.Panic(err)
		}
		value := reflect.ValueOf(config)

		if value.Type().Kind() != reflect.Ptr {
			log.Logger.Panic("Pointer type required")
		}

		value = value.Elem()

		for i := 0; i < value.NumField(); i++ {
			v := value.Field(i)

			if v.IsNil() {
				name := value.Type().Field(i).Name
				log.Logger.Debugf("%s config not found, check if [%s] contains valid section [%s].", name, viper.ConfigFileUsed(), strings.ToLower(name))
			}
			f := v.Interface()
			initCB, ok := f.(OnChangeCallback)
			if ok {
				err := initCB.Change()
				if err != nil {
					log.Logger.Panic(err)
				}
				continue
			}
		}

	})
}
