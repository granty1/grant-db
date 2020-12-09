package main

import (
	"grant-db/config"
	"grant-db/kv"
	"grant-db/server"
	"log"
)

var (
	storage kv.Storage

	srv *server.Server
	cfg *config.Config
)

func main() {
	//TODO 参数解析 -> flags ....

	//TODO 注册存储引擎

	//TODO 注册数据统计Metrics

	//TODO 加载配置，初始目录结构
	cfg = config.InitConfig()
	//TODO 设置全局参数

	//TODO 初始化日志模块

	//TODO 初始化堆性能追踪器

	//TODO 初始化tracing

	//TODO 启动Metrics

	//TODO 创建存储引擎
	createStore()
	//TODO 创建Server
	createServer()

	//TODO 监听退出信号

	//TODO 启动服务器
	runServer()
}

func createStore() {
	storage = kv.NewStorage()
}

func createServer() {
	//TODO get config
	driver := server.NewGrantDBDriver(storage)
	//TODO create server
	srv = server.NewServer(cfg, driver)
	//TODO clean domain and storage if create server error
}

func runServer() {
	log.Fatalln(srv.Run())
}
