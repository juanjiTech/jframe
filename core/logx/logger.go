package logx

import (
	"log"
	"os"
	"sync"

	"github.com/juanjiTech/jframe/conf"
	"github.com/juanjiTech/jframe/pkg/clsLog"
	tencentcloud_cls_sdk_go "github.com/tencentcloud/tencentcloud-cls-sdk-go"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	lumberjackV2 "gopkg.in/natefinch/lumberjack.v2"
)

var (
	inited bool
	mu     sync.Mutex
)

func NameSpace(name string) *zap.SugaredLogger {
	return zap.S().Named(name)
}

func getLogWriter() zapcore.WriteSyncer {
	if conf.Get().Log.LogPath == "" {
		log.Fatalln("LogPath 未设置")
	}
	lj := &lumberjackV2.Logger{
		Filename:   conf.Get().Log.LogPath,
		MaxSize:    5,
		MaxBackups: 5,
		MaxAge:     30,
		Compress:   true,
	}
	return zapcore.AddSync(lj)
}

func getEncoder() zapcore.Encoder {
	encoderConfig := zap.NewDevelopmentEncoderConfig()
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	encoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder
	return zapcore.NewConsoleEncoder(encoderConfig)
}

func PreInit() {
	encoder := getEncoder()
	core := zapcore.NewCore(encoder, os.Stdout, zap.DebugLevel)
	options := []zap.Option{
		zap.AddCaller(),
		zap.AddStacktrace(zap.ErrorLevel),
	}
	zap.ReplaceGlobals(zap.New(core, options...))
}

func Init(level zapcore.LevelEnabler) {
	mu.Lock()
	defer mu.Unlock()

	if inited {
		zap.S().Debug("logx.Init skipped: already initialized in this package")
		return
	}

	if zap.L().Core() != zapcore.NewNopCore() {
		zap.S().Debug("logx.Init skipped: zap already initialized externally")
		inited = true
		return
	}

	writeSyncer := zapcore.AddSync(os.Stdout)
	if conf.Get().Log.LogPath != "" {
		writeSyncer = zapcore.NewMultiWriteSyncer(writeSyncer, getLogWriter())
	}

	encoder := getEncoder()
	core := zapcore.NewCore(encoder, writeSyncer, level)
	options := []zap.Option{
		zap.AddCaller(),
		zap.AddStacktrace(zap.ErrorLevel),
	}

	if CLSConfig := conf.Get().Log.CLS; CLSConfig.Endpoint != "" {
		producerConfig := tencentcloud_cls_sdk_go.GetDefaultAsyncProducerClientConfig()
		producerConfig.Endpoint = CLSConfig.Endpoint
		producerConfig.AccessKeyID = CLSConfig.AccessKey
		producerConfig.AccessKeySecret = CLSConfig.AccessToken

		hook, err := clsLog.NewZapHook(producerConfig, level, CLSConfig.TopicID)
		if err != nil {
			zap.S().Fatal(err)
		}
		options = append(options, zap.Hooks(hook.Hook))
		zap.S().Info("CLS Hook Init Success")
	}

	zap.ReplaceGlobals(zap.New(core, options...))
	inited = true
}
