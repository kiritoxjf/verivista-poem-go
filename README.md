# verivista-poem-go
每日诗词更新
> 每日0、12时获取诗词数据写入数据库

## 配置文件
```json5
// config.json
{
  "token": "", // 今日诗词开放API获取
  "db": {
    "ip_addr": "",
    "port": "",
    "driver": "mysql",
    "user": "",
    "pass": "",
    "name": ""
  }
}
```

## docker-compose
通过环境变量获取日志文件与配置文件的绝对路径，以便进行文件映射，持久化两文件
```yaml
  poem:
    image: alpine:latest
    network_mode: host
    container_name: poem
    environment:
      - POEM_LOG_PATH=/poem/log/poem.log
      - POEM_CONFIG_PATH=/poem/config/config.json
    volumes:
      - /{your_path}/verivista-poem-go:/poem/verivista-poem-go
      - /{your_path}/config.json:/poem/config/config.json
      - /{your_path}/poem.log:/poem/log/poem.log
    command: ["/poem/verivista-poem-go"]
```