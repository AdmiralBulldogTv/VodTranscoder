package main

import (
	"context"
	"os"
	"os/signal"
	"runtime"
	"runtime/debug"
	"strconv"
	"syscall"
	"time"

	"github.com/AdmiralBulldogTv/VodTranscoder/src/configure"
	"github.com/AdmiralBulldogTv/VodTranscoder/src/global"
	"github.com/AdmiralBulldogTv/VodTranscoder/src/health"
	"github.com/AdmiralBulldogTv/VodTranscoder/src/monitoring"
	"github.com/AdmiralBulldogTv/VodTranscoder/src/svc/mongo"
	"github.com/AdmiralBulldogTv/VodTranscoder/src/svc/prometheus"
	"github.com/AdmiralBulldogTv/VodTranscoder/src/svc/redis"
	"github.com/AdmiralBulldogTv/VodTranscoder/src/svc/rmq"
	"github.com/AdmiralBulldogTv/VodTranscoder/src/transcoder"

	"github.com/bugsnag/panicwrap"
	"github.com/sirupsen/logrus"
)

var (
	Version = "development"
	Unix    = ""
	Time    = "unknown"
	User    = "unknown"
)

func init() {
	debug.SetGCPercent(2000)
	if i, err := strconv.Atoi(Unix); err == nil {
		Time = time.Unix(int64(i), 0).Format(time.RFC3339)
	}
}

func main() {
	config := configure.New()

	exitStatus, err := panicwrap.BasicWrap(func(s string) {
		logrus.Error(s)
	})
	if err != nil {
		logrus.Error("failed to setup panic handler: ", err)
		os.Exit(2)
	}

	if exitStatus >= 0 {
		os.Exit(exitStatus)
	}

	if !config.NoHeader {
		logrus.Info("Vods Transcoder")
		logrus.Infof("Version: %s", Version)
		logrus.Infof("build.Time: %s", Time)
		logrus.Infof("build.User: %s", User)
	}

	logrus.Debug("MaxProcs: ", runtime.GOMAXPROCS(0))

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)

	c, cancel := context.WithCancel(context.Background())

	gCtx := global.New(c, config)

	{
		ctx, cancel := context.WithTimeout(gCtx, time.Second*15)
		redisInst, err := redis.New(ctx, redis.SetupOptions{
			Username:   gCtx.Config().Redis.Username,
			Password:   gCtx.Config().Redis.Password,
			MasterName: gCtx.Config().Redis.MasterName,
			Database:   gCtx.Config().Redis.Database,
			Addresses:  gCtx.Config().Redis.Addresses,
			Sentinel:   gCtx.Config().Redis.Sentinel,
		})
		cancel()
		if err != nil {
			logrus.WithError(err).Fatal("failed to connect to redis")
		}

		gCtx.Inst().Redis = redisInst
	}

	{
		ctx, cancel := context.WithTimeout(gCtx, time.Second*15)
		mongoInst, err := mongo.New(ctx, mongo.SetupOptions{
			URI:      gCtx.Config().Mongo.URI,
			Database: gCtx.Config().Mongo.Database,
			Direct:   gCtx.Config().Mongo.Direct,
			Indexes:  []mongo.IndexRef{},
		})
		cancel()
		if err != nil {
			logrus.WithError(err).Fatal("failed to connect to mongo")
		}

		gCtx.Inst().Mongo = mongoInst
	}

	{
		gCtx.Inst().Prometheus = prometheus.New(prometheus.SetupOptions{
			Labels: prometheus.LabelsFromKeyValue(gCtx.Config().Monitoring.Labels),
		})
	}

	{
		ctx, cancel := context.WithTimeout(gCtx, time.Second*15)
		rmqInst, err := rmq.New(ctx, rmq.SetupOptions{
			URI:                     gCtx.Config().RMQ.URI,
			TranscoderTaskQueueName: gCtx.Config().RMQ.TranscoderTaskQueue,
			NoopQueue:               "noop-queue",
		})
		cancel()
		if err != nil {
			logrus.WithError(err).Fatal("failed to connect to rmq")
		}

		gCtx.Inst().RMQ = rmqInst
	}

	dones := []<-chan struct{}{transcoder.New(gCtx)}
	if gCtx.Config().Health.Enabled {
		dones = append(dones, health.New(gCtx))
	}
	if gCtx.Config().Monitoring.Enabled {
		dones = append(dones, monitoring.New(gCtx))
	}

	logrus.Info("running")

	done := make(chan struct{})
	go func() {
		<-sig
		cancel()
		go func() {
			select {
			case <-time.After(time.Minute):
			case <-sig:
			}
			logrus.Fatal("force shutdown")
		}()

		for _, d := range dones {
			<-d
		}

		logrus.Info("shutting down")
		close(done)
	}()

	<-done

	logrus.Info("shutdown")
	os.Exit(0)
}
