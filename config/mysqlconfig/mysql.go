package mysqlconfig

import (
	"database/sql"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"github.com/zhouchang2017/toolkit/log"
	"net/url"
	"strings"
)

var (
	instances = map[string]*sql.DB{}
)

func Get(dbHandler string) (*sql.DB, error) {
	if db, ok := instances[dbHandler]; ok {
		return db, nil
	}
	return nil, fmt.Errorf("DBHandler[%s] not found", dbHandler)
}

type Configs map[string]Config

func (c *Configs) Init() error {
	if c == nil {
		return nil
	}
	for name, config := range *c {
		db, err := config.Open()
		if err != nil {
			log.Logger.Errorf("db [%s] open conn err:%s", name, err.Error())
		}
		instances[name] = db
	}
	return nil
}

func (c *Configs) Close() {
	for name, db := range instances {
		if err := db.Close(); err != nil {
			log.Logger.Errorf("db [%s] close err:%s", name, err.Error())
		}
	}
}

type Config struct {
	Host        string
	Port        string
	DB          string
	Username    string
	Password    string
	Charset     string
	MaxOpenConn int
	MaxIdleConn int
	Timeout     int
}

func (c Config) String() string {
	var buf strings.Builder
	if c.Username != "" && c.Password != "" {
		buf.WriteString(url.UserPassword(c.Username, c.Password).String())
		buf.WriteByte('@')
	}
	if c.Port == "" {
		c.Port = "3306"
	}
	if c.Host == "" {
		c.Host = "127.0.0.1"
	}
	buf.WriteString("(")
	buf.WriteString(c.Host)
	buf.WriteString(":")
	buf.WriteString(c.Port)
	buf.WriteString(")")
	if c.DB != "" {
		buf.WriteString("/")
		buf.WriteString(c.DB)
	}
	values := url.Values{}
	if c.Charset == "" {
		c.Charset = "utf8mb4"
	}
	values.Add("charset", c.Charset)
	values.Add("parseTime", "True")
	values.Add("loc", "Local")

	buf.WriteString("?")
	buf.WriteString(values.Encode())
	defer buf.Reset()
	return buf.String()
}

func (c Config) Open() (*sql.DB, error) {
	db, err := sql.Open("mysql", c.String())
	if err != nil {
		return nil, err
	}
	if c.MaxOpenConn == 0 {
		c.MaxOpenConn = 20
	}
	if c.MaxIdleConn == 0 {
		c.MaxIdleConn = 20
	}
	db.SetMaxOpenConns(c.MaxOpenConn)
	db.SetMaxIdleConns(c.MaxIdleConn)
	return db, db.Ping()
}
