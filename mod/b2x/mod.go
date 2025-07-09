package b2x

import (
	"context"
	"errors"
	"github.com/Backblaze/blazer/b2"
	"github.com/juanjiTech/jframe/core/kernel"
	"sync"
)

var _ kernel.Module = (*Mod)(nil)

type Config struct {
	BucketKeyID string `yaml:"bucketKeyId" mapstructure:"bucketKeyId"`
	BucketKey   string `yaml:"bucketKey" mapstructure:"bucketKey"`
	BucketName  string `yaml:"bucketName" mapstructure:"bucketName"`
}

type Mod struct {
	kernel.UnimplementedModule // 请为所有Module引入UnimplementedModule

	config Config

	cancel context.CancelFunc
}

func (m *Mod) Config() any {
	return &m.config
}

func (m *Mod) Name() string {
	return "b2"
}

func (m *Mod) PreInit(hub *kernel.Hub) error {
	ctx, cancel := context.WithCancel(context.Background())
	m.cancel = cancel
	b2Conf := m.config
	b2Client, err := b2.NewClient(ctx, b2Conf.BucketKeyID, b2Conf.BucketKey)
	if err != nil {
		hub.Log.Error(err)
		cancel()
		return err
	}
	b2Bucket, err := b2Client.Bucket(ctx, b2Conf.BucketName)
	if err != nil {
		hub.Log.Error(err)
		cancel()
		return err
	}
	hub.Map(&b2Client, &b2Bucket)
	return nil
}

func (m *Mod) Init(hub *kernel.Hub) error {
	var b2Client *b2.Client
	if hub.Load(&b2Client) != nil {
		return errors.New("can't load b2 client from kernel")
	}

	var b2Bucket *b2.Bucket
	if hub.Load(&b2Bucket) != nil {
		return errors.New("can't load b2 bucket from kernel")
	}
	return nil
}

func (m *Mod) Stop(wg *sync.WaitGroup, _ context.Context) error {
	defer wg.Done()
	m.cancel()
	return nil
}
