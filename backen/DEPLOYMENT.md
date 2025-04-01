# 许可证管理系统部署指南

## 系统要求
- Go 1.18+
- SQLite3 (嵌入式数据库)
- 或 MySQL 5.7+ (如需使用MySQL)

## 1. 获取代码
```bash
git clone https://github.com/your-repo/mql_License.git
cd mql_License/backen/backend
```

## 2. 配置数据库
### SQLite配置(默认)
无需额外配置，系统会自动创建`data/license.db`文件

### MySQL配置(可选)
1. 修改`config.yaml`:
```yaml
database:
  driver: "mysql"
  dsn: "username:password@tcp(127.0.0.1:3306)/license_db?charset=utf8mb4&parseTime=True&loc=Local"
```

## 3. 安装依赖
```bash
go mod download
```

## 4. 运行程序
### 开发模式
```bash
go run cmd/main.go
```

### 生产模式
```bash
# 构建
go build -o license-manager cmd/main.go

# 运行
./license-manager
```

## 5. 系统配置
主要配置项(`config.yaml`):
```yaml
server:
  port: 80  # 服务端口
  jwt_secret: "your_jwt_secret" # JWT密钥

database:
  driver: "sqlite3" # 或 "mysql"
  dsn: "data/license.db" # SQLite文件路径或MySQL连接字符串
```

## 6. 系统服务管理(生产环境)
创建systemd服务文件`/etc/systemd/system/license-manager.service`:
```
[Unit]
Description=License Manager Service
After=network.target

[Service]
User=root
WorkingDirectory=/path/to/mql_License/backen/backend
ExecStart=/path/to/mql_License/backen/backend/license-manager
Restart=always

[Install]
WantedBy=multi-user.target
```

启用服务:
```bash
sudo systemctl daemon-reload
sudo systemctl start license-manager
sudo systemctl enable license-manager
```

## 7. 验证部署
```bash
curl http://localhost:3001/api/v1/health
```
应返回`{"status":"ok"}`

## 8. 维护命令
- 数据库迁移: `go run cmd/main.go -migrate`
- 查看日志: `journalctl -u license-manager -f`
