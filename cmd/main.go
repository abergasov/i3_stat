package main

import (
	"flag"
	"fmt"
	"i3_stat/internal/config"
	"i3_stat/internal/logger"
	"i3_stat/internal/routes"
	samplerService "i3_stat/internal/service/sampler"
	"i3_stat/internal/storage/database"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.uber.org/zap"
)

var (
	confFile = flag.String("config", "configs/app_conf.yml", "Configs file path")
	appHash  = os.Getenv("")
)

func main() {
	flag.Parse()
	appLog, err := logger.NewAppLogger(appHash)
	if err != nil {
		log.Fatalf("unable to create logger: %s", err)
	}
	appLog.Info("app starting", zap.String("conf", *confFile))
	appConf, err := config.InitConf(*confFile)
	if err != nil {
		appLog.Fatal("unable to init config", err, zap.String("config", *confFile))
	}

	appLog.Info("init services")
	service := samplerService.InitService(appLog, appConf.CryptocompareAPIKey)

	appLog.Info("init http service")
	appHTTPServer := routes.InitAppRouter(appLog, service, fmt.Sprintf(":%d", appConf.AppPort))
	defer func() {
		if err = appHTTPServer.Stop(); err != nil {
			appLog.Fatal("unable to stop http service", err)
		}
	}()
	go func() {
		if err = appHTTPServer.Run(); err != nil {
			appLog.Fatal("unable to start http service", err)
		}
	}()

	// register app shutdown
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c // This blocks the main thread until an interrupt is received
}

func getDBConnect(log logger.AppLogger, cnf *config.DBConf, migratesFolder string) (*database.DBConnect, error) {
	for i := 0; i < 5; i++ {
		dbConnect, err := database.InitDBConnect(cnf, migratesFolder)
		if err == nil {
			return dbConnect, nil
		}
		time.Sleep(time.Duration(i) * time.Second * 5)
		log.Error("can't connect to db", err, zap.Int("attempt", i))
	}
	return nil, fmt.Errorf("can't connect to db")
}
