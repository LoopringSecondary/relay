package gateway_test

import (
	"testing"
	//"github.com/Loopring/relay/test"
	"fmt"
	"github.com/Loopring/relay/cache"
	"github.com/Loopring/relay/config"
	"github.com/Loopring/relay/log"
	"github.com/Loopring/relay/market"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"time"
)

func TestWalletServiceImpl_GetTrend(t *testing.T) {
	lc := config.LogOptions{}
	zapOpt := zap.Config{}
	zapOpt.Level = zap.NewAtomicLevelAt(zapcore.DebugLevel)
	zapOpt.Development = true
	zapOpt.DisableStacktrace = false
	zapOpt.Encoding = "console"
	zapOpt.EncoderConfig = zapcore.EncoderConfig{
		MessageKey:    "msg",
		TimeKey:       "ts",
		LevelKey:      "level",
		StacktraceKey: "trace",
		EncodeLevel: func(level zapcore.Level, encoder zapcore.PrimitiveArrayEncoder) {

		},
		EncodeTime: func(time time.Time, encoder zapcore.PrimitiveArrayEncoder) {

		}}
	zapOpt.OutputPaths = []string{"zap.log", "stderr"}
	lc.ZapOpts = zapOpt

	log.Initialize(lc)
	fmt.Println("xxxxxx")
	//redisCache := &redis.RedisCacheImpl{}
	cfg := config.RedisOptions{}
	cfg.Host = "127.0.0.1"
	cfg.Port = "6379"
	cfg.Password = "aaaaaa"
	cfg.IdleTimeout = 20
	cfg.MaxIdle = 2
	cfg.MaxActive = 5
	//redisCache.Initialize(cfg)
	cache.NewCache(cfg)

	//redisCache.HMSet()
	//fmt.Println(redisCache.Get("1234"))
	c := market.NewCollector(true)
	c.Start()
	fmt.Println(c.GetTickers("LRC-WETH"))
	time.Sleep(1 * time.Second)
	fmt.Println(c.GetTickers("LRC-WETH"))
	time.Sleep(1 * time.Second)
	fmt.Println(c.GetTickers("FOO-BAR"))
	time.Sleep(1 * time.Second)
	fmt.Println(c.GetTickers("LRC-WETH"))
	time.Sleep(1 * time.Second)
	fmt.Println(c.GetTickers("LRC-WETH"))
	time.Sleep(1 * time.Second)
	fmt.Println(c.GetTickers("LRC-WETH"))
	time.Sleep(1 * time.Second)
	fmt.Println(c.GetTickers("LRC-WETH"))
	time.Sleep(1 * time.Second)
	fmt.Println(c.GetTickers("LRC-WETH"))

}
