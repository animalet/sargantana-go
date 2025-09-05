package main

import (
	"os"

	"github.com/animalet/sargantana-go/config"
	"github.com/animalet/sargantana-go/controller"
	"github.com/animalet/sargantana-go/server"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
)

func main() {
	server.AddControllerType("auth", controller.NewAuthController)
	server.AddControllerType("static", controller.NewStaticController)
	server.AddControllerType("template", controller.NewTemplateController)

	cfg, err := config.LoadYaml[config.Config]("./config.yaml")
	if err != nil {
		panic(err)
	}

	dbConfig, err := config.LoadPartial[DatabaseConfig]("database", cfg)

	if err != nil {
		log.Fatal().Err(err).Msg("Failed to load database configuration")
		os.Exit(1)
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

	server.AddControllerType("blog", NewBlogController(database))

	sargantana, err := server.NewServer(cfg)

	if err != nil {
		log.Fatal().Err(err).Msg("Failed to create server")
		os.Exit(1)
	}

	err = sargantana.StartAndWaitForSignal()
	if err != nil {
		log.Fatal().Err(err).Msg("Server error")
	}
}

type (
	BlogController struct {
		config   *BlogConfig
		database interface{} // Replace with actual database type
	}
	BlogConfig struct {
		FeedUrl      string `yaml:"feed_url"`
		PostUrl      string `yaml:"post_url"`
		AdminAreaUrl string `yaml:"admin_area_url"`
	}
	DatabaseConfig struct {
		Host     string `yaml:"host"`
		Port     uint16 `yaml:"port"`
		Database string `yaml:"database"`
		User     string `yaml:"user"`
		Password string `yaml:"password"`
	}
)

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

func (b BlogConfig) Validate() error {
	if b.FeedUrl == "" {
		return errors.New("feed_url must be set and non-empty")
	}
	if b.PostUrl == "" {
		return errors.New("post_url must be set and non-empty")
	}
	if b.AdminAreaUrl == "" {
		return errors.New("admin_area_url must be set and non-empty")
	}
	return nil
}

func (b *BlogController) Bind(engine *gin.Engine, loginMiddleware gin.HandlerFunc) {
	api := engine.Group("/api")
	{
		api.GET(b.config.FeedUrl, b.getFeed)
		api.POST(b.config.PostUrl, b.createPost)
		api.GET(b.config.PostUrl+"/:id", b.getPost)
		api.DELETE(b.config.PostUrl+"/:id", b.deletePost)
		api.GET(b.config.AdminAreaUrl, loginMiddleware, b.adminArea)
	}
}

func (b *BlogController) Close() error { return nil }

func NewBlogController(db any) controller.Constructor {
	return func(configData config.ControllerConfig, _ config.ServerConfig) (controller.IController, error) {
		cfg, err := config.UnmarshalTo[BlogConfig](configData)
		if err != nil {
			return nil, err
		}
		return &BlogController{config: cfg, database: db}, nil
	}
}

func (b *BlogController) getPost(c *gin.Context) {

}

func (b *BlogController) createPost(c *gin.Context) {
	// Implementation here
}

func (b *BlogController) deletePost(c *gin.Context) {
	// Implementation here
}

func (b *BlogController) getFeed(c *gin.Context) {
	// Implementation here
}

func (b *BlogController) adminArea(c *gin.Context) {
	// Implementation here
}
