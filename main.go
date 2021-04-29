package main

import (
	"context"
	"flag"
	"fmt"
	"fuckdb/config"
	"fuckdb/routers"
	"fuckdb/routers/middleware"
	"io/ioutil"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/gin-contrib/static"
	"github.com/gin-gonic/gin"
	_ "github.com/go-sql-driver/mysql"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

func main() {

	g := gin.Default()
	//Use frontend static
	g.Use(static.Serve("/", static.LocalFile("./frontend/dist", false)))
	// LoggerWithFormatter middleware
	// By default gin.DefaultWriter = os.Stdout
	g.Use(gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
		b, _ := ioutil.ReadAll(param.Request.Body)
		// your custom format
		return fmt.Sprintf("StatusCode:%d--%s-ClientIp:%s - TimeStamp:[%s] \"ReqMethod:%s--"+
			"API:%s--ReqProto%s--CostTime:%s \"UserAgent:%s-\" -Error:%s\"\n",
			param.StatusCode,
			string(b),
			param.ClientIP,
			param.TimeStamp.Format(time.RFC1123),
			param.Method,
			param.Path,
			param.Request.Proto,
			param.Latency,
			param.Request.UserAgent(),
			param.ErrorMessage,
		)
	}))

	//Add middleware in this slice
	var middlewares = []gin.HandlerFunc{middleware.NoCache, middleware.Secure}

	routers.Load(
		g,
		middlewares...,
	)

	// init config
	if err := config.InitConfig(""); err != nil {
		logrus.Errorln("init config error:%s", err)
		panic(err)
	}
	logrus.Infoln("config init success")
	var host = viper.GetString("server.host")
	var port = viper.GetString("server.port")
	fmt.Print(host + ":" + port)
	var addr = flag.String("server addr", host+":"+port, "server addr")
	// run server
	srv := http.Server{
		Addr:    *addr,
		Handler: g,
	}
	processd := make(chan struct{})
	go func() {
		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt)
		<-c

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := srv.Shutdown(ctx); err != nil {
			logrus.Errorln("server shutdown failed,err:", err)
		}
		logrus.Infoln("server shutdown gracefully")
		close(processd)
	}()

	err := srv.ListenAndServe()
	fmt.Println(*addr)
	fmt.Println("server successly")
	if err != http.ErrServerClosed {
		logrus.Errorln("server not shutdown gracefully,err:", err)
	}

	<-processd
}
