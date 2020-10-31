# golang develop tool kit

## Config
##### Example

入口文件`main.go`
```go
package main

var config *Conf

type Conf struct {
	Logs  *logconfig.Configs
	DB    *mysqlconfig.Configs
	Redis *redisconfig.Config
}

func main() {

	conf = &Conf{}
	ctx := context.Background()

	if err := config.Init(ctx, conf, "", ""); err != nil {
		panic(err)
	}

	defer config.LoadOnCloseCallbacks(conf)
}

```

config.yml
```yaml
logs:
  app:
    driver: zap
    stdout: true
    formatter: plain
    level: info
    enable_file_line: true
    path: /tmp/app.accesslog
    default: true
mysql:
  db1:
    host: 127.0.0.1
    port: 3306
    db: dbname
    username: root
    password: 12345678
  db2:
    host: 127.0.0.1
    port: 3306
    db: dbname
    username: root
    password: 12345678
redis:
  redis1:
    addr: 127.0.0.1:6397
```

### Log
目前仅提供 `zap` 日志

