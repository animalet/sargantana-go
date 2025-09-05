package main

import (
	"github.com/animalet/sargantana-go/config"
	"github.com/animalet/sargantana-go/controller"
	"github.com/animalet/sargantana-go/examples/blog_example/blog"
	"github.com/animalet/sargantana-go/server"
	"github.com/jackc/pgx"
	"github.com/pkg/errors"
)

type DatabaseConfig struct {
	Host     string `yaml:"host"`
	Port     uint16 `yaml:"port"`
	Database string `yaml:"database"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
}

func (d DatabaseConfig) Validate() error {
	if d.Host == "" {
		return errors.New("host must be set and non-empty")
	}
	if d.Port == 0 {
		return errors.New("port must be set and non-zero")
	}
	if d.Database == "" {
		return errors.New("database must be set and non-empty")
	}
	if d.User == "" {
		return errors.New("user must be set and non-empty")
	}
	if d.Password == "" {
		return errors.New("password must be set and non-empty")
	}
	return nil
}

func main() {
	server.AddControllerType("auth", controller.NewAuthController)
	server.AddControllerType("static", controller.NewStaticController)
	server.AddControllerType("template", controller.NewTemplateController)

	cfg, err := config.ReadConfig("./config.yaml")
	if err != nil {
		panic(err)
	}

	dbConfig, err := config.LoadConfig[DatabaseConfig]("database", cfg)
	if err != nil {
		panic(err)
	}

	database, err := pgx.Connect(pgx.ConnConfig{
		Host:     dbConfig.Host,
		Port:     dbConfig.Port,
		Database: dbConfig.Database,
		User:     dbConfig.User,
		Password: dbConfig.Password,
	})

	if err != nil {
		panic(err)
	}
	defer database.Close()

	server.AddControllerType("blog", blog.NewBlogController(database))

	sargantana, err := server.NewServer(cfg)

	if err != nil {
		panic(err)
	}

	err = sargantana.StartAndWaitForSignal()
	if err != nil {
		panic(err)
	}
}
