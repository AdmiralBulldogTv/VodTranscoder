package redis

import (
	"context"
	"sync"

	"github.com/AdmiralBulldogTv/VodTranscoder/src/instance"
	"github.com/go-redis/redis/v8"
	"github.com/sirupsen/logrus"
)

type RedisInst struct {
	client  *redis.Client
	sub     *redis.PubSub
	subsMtx sync.Mutex
	subs    map[string][]*redisSub
}

type SetupOptions struct {
	Username   string
	Password   string
	MasterName string
	Database   int

	Addresses []string
	Sentinel  bool
}

func New(ctx context.Context, opts SetupOptions) (instance.Redis, error) {
	if len(opts.Addresses) == 0 {
		logrus.Fatal("you must provide at least one redis address")
	}

	var rc *redis.Client
	if opts.Sentinel {
		rc = redis.NewFailoverClient(&redis.FailoverOptions{
			MasterName:       opts.MasterName,
			SentinelAddrs:    opts.Addresses,
			SentinelUsername: opts.Username,
			SentinelPassword: opts.Password,
			Username:         opts.Username,
			Password:         opts.Password,
			DB:               opts.Database,
		})
	} else {
		rc = redis.NewClient(&redis.Options{
			Addr:     opts.Addresses[0],
			Username: opts.Username,
			Password: opts.Password,
			DB:       opts.Database,
		})
	}

	if err := rc.Ping(ctx).Err(); err != nil {
		return nil, err
	}

	inst := &RedisInst{
		client: rc,
		sub:    rc.Subscribe(context.Background()),
		subs:   map[string][]*redisSub{},
	}
	go func() {
		defer func() {
			if err := recover(); err != nil {
				logrus.WithField("err", err).Fatal("panic in subs")
			}
		}()
		ch := inst.sub.Channel()
		var msg *redis.Message
		for {
			msg = <-ch
			payload := msg.Payload // dont change we want to copy the memory due to concurrency.
			inst.subsMtx.Lock()
			for _, s := range inst.subs[msg.Channel] {
				select {
				case s.ch <- payload:
				default:
					logrus.Warn("channel blocked dropping message: ", msg.Channel)
				}
			}
			inst.subsMtx.Unlock()
		}
	}()

	return inst, nil
}

type redisSub struct {
	ch chan string
}

// Subscribe to a channel on Redis
func (r *RedisInst) Subscribe(ctx context.Context, ch chan string, subscribeTo ...string) {
	r.subsMtx.Lock()
	defer r.subsMtx.Unlock()
	localSub := &redisSub{ch}
	for _, e := range subscribeTo {
		if _, ok := r.subs[e]; !ok {
			_ = r.sub.Subscribe(ctx, e)
		}
		r.subs[e] = append(r.subs[e], localSub)
	}

	go func() {
		<-ctx.Done()
		r.subsMtx.Lock()
		defer r.subsMtx.Unlock()
		for _, e := range subscribeTo {
			for i, v := range r.subs[e] {
				if v == localSub {
					if i != len(r.subs[e])-1 {
						r.subs[e][i] = r.subs[e][len(r.subs[e])-1]
					}
					r.subs[e] = r.subs[e][:len(r.subs[e])-1]
					if len(r.subs[e]) == 0 {
						delete(r.subs, e)
						if err := r.sub.Unsubscribe(context.Background(), e); err != nil {
							logrus.WithError(err).Error("failed to unsubscribe")
						}
					}
					break
				}
			}
		}
	}()
}

func (r *RedisInst) Ping(ctx context.Context) error {
	return r.client.Ping(ctx).Err()
}
