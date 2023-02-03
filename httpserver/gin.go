package httpserver

import (
	"context"
	"fmt"
	"net/http"
	"time"

	gincors "github.com/gin-contrib/cors"
	gingzip "github.com/gin-contrib/gzip"
	"github.com/gin-contrib/pprof"
	"github.com/gin-contrib/requestid"
	"github.com/gin-gonic/gin"
)

func NewRouter(serviceRunMode string, profile bool) *gin.Engine {
	gin.SetMode(serviceRunMode)
	r := gin.New()

	r.Use(customLogger())

	r.Use(
		cors(),
		gin.Recovery(),
		requestid.New(),
		gingzip.Gzip(gingzip.DefaultCompression),
	)

	if profile {
		pprof.Register(r)
	}

	return r
}

func Run(ctx context.Context, router http.Handler, servicePort string, graceful time.Duration) error {
	fmt.Printf("========|||======== \n http server listen on %s \n========|||========\n", servicePort)

	cancelCtx, cancel := context.WithCancel(ctx)
	srv := http.Server{
		Addr:    ":" + servicePort,
		Handler: router,
	}

	var err error

	go func() {
		if err = srv.ListenAndServe(); err != nil {
			cancel()
		}
	}()

	<-cancelCtx.Done()

	if err != nil {
		return err
	}

	ctxShutdown, cancelShutdown := context.WithTimeout(context.Background(), graceful)
	defer cancelShutdown()
	fmt.Println("====++++    http server shutdown    ++++====")

	return srv.Shutdown(ctxShutdown)
}

func cors() gin.HandlerFunc {
	config := gincors.DefaultConfig()
	config.AllowAllOrigins = true
	config.AddAllowMethods("POST", "GET", "PUT", "DELETE")
	config.AllowCredentials = true

	return gincors.New(config)
}

func customLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		startTime := time.Now()

		c.Next()

		castTime := time.Since(startTime)
		fmt.Printf("统计耗时：---- %d %s %s (%s) %.2fms ----\n",
			c.Writer.Status(),
			c.Request.Method,
			c.Request.URL.Path,
			c.ClientIP(),
			float64(castTime.Microseconds())/1000, // 这里用 微秒 是为了保留精度.
		)
	}
}
