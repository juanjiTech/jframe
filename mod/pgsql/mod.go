package pgsql

import (
	"errors"
	"fmt"
	"github.com/juanjiTech/jframe/core/kernel"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var _ kernel.Module = (*Mod)(nil)

type Config struct {
	Host     string `yaml:"host" mapstructure:"host"`
	PORT     string `yaml:"port" mapstructure:"port"`
	User     string `yaml:"user" mapstructure:"user"`
	Password string `yaml:"password" mapstructure:"password"`
	Name     string `yaml:"name" mapstructure:"name"`
	SSLMode  string `yaml:"sslMode" mapstructure:"sslMode"`
}

type Mod struct {
	kernel.UnimplementedModule // 请为所有Module引入UnimplementedModule

	config Config
}

func (m *Mod) Name() string {
	return "postgres"
}

func (m *Mod) Config() any { return &m.config }

func (m *Mod) PreInit(hub *kernel.Hub) error {
	c := m.config
	if c.PORT == "" {
		c.PORT = "5432"
	}
	dsn := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s",
		c.User, c.Password, c.Host, c.PORT, c.Name, c.SSLMode)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return err
	}

	hub.Log.Info("postgres init success")
	hub.Map(&db)
	return nil
}

func (m *Mod) Init(hub *kernel.Hub) error {
	// check if inject success
	var db *gorm.DB
	if hub.Load(&db) != nil {
		return errors.New("can't load gorm from kernel")
	}

	var tables []string
	result := db.Raw("SELECT table_name FROM information_schema.tables WHERE table_schema = 'public'").Scan(&tables)
	if result.Error != nil {
		return result.Error
	}
	return nil
}
