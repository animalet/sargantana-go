package blog

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/animalet/sargantana-go/pkg/config"
	"github.com/animalet/sargantana-go/pkg/controller"
	"github.com/animalet/sargantana-go/pkg/server"
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
)

type (
	Controller struct {
		config   *Config
		database *pgxpool.Pool
	}
	Config struct {
		FeedPath      string `yaml:"feed_path"`
		PostPath      string `yaml:"post_path"`
		AdminAreaPath string `yaml:"admin_area_path"`
	}
)

func (b Config) Validate() error {
	if b.FeedPath == "" {
		return errors.New("feed_path must be set and non-empty")
	}
	if b.PostPath == "" {
		return errors.New("post_url must be set and non-empty")
	}
	if b.AdminAreaPath == "" {
		return errors.New("admin_area_url must be set and non-empty")
	}
	return nil
}

func (b *Controller) Bind(engine *gin.Engine) {
	engine.GET(b.config.FeedPath, b.getFeed)
	engine.POST(b.config.PostPath, b.createPost)
	engine.GET(b.config.PostPath+"/:id", b.getPost)
	engine.DELETE(b.config.PostPath+"/:id", b.deletePost)
	engine.GET(b.config.AdminAreaPath, controller.LoginFunc, b.adminArea)
}

func (b *Controller) Close() error { return nil }

func NewBlogController(db *pgxpool.Pool) server.ControllerFactory {
	return func(configData config.ModuleRawConfig, _ server.ControllerContext) (server.IController, error) {
		cfg, err := config.Unmarshal[Config](configData)
		if err != nil {
			return nil, err
		}

		tag, err := db.Exec(context.Background(), `CREATE TABLE IF NOT EXISTS posts (
        id SERIAL PRIMARY KEY,
        title TEXT NOT NULL,
        content TEXT NOT NULL,
        publication_date TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
        owner TEXT NOT NULL
    )`)

		if tag.RowsAffected() > 0 {
			log.Info().Msg("Posts table created")
		}

		if err != nil {
			return nil, errors.Wrap(err, "failed to create posts table")
		}

		return &Controller{config: cfg, database: db}, nil
	}
}

type post struct {
	Id              int    `form:"id" json:"id"`
	Title           string `form:"title" json:"title"`
	Content         string `form:"content" json:"content"`
	PublicationDate time.Time
	Owner           string
}

const (
	articleTemplate = "articles.html"
	adminTemplate   = "admin.html"
)

func (b *Controller) getPost(c *gin.Context) {
	p := post{}
	err := b.database.QueryRow(context.Background(), "SELECT id, title, content, publication_date, owner FROM posts WHERE id=$1", c.Param("id")).
		Scan(&p.Id, &p.Title, &p.Content, &p.PublicationDate, &p.Owner)
	if isDBError(c, err) {
		return
	}

	c.HTML(200, articleTemplate, gin.H{
		"user": b.getUserId(c),
		"feed": []post{p},
	})
}

func (b *Controller) createPost(c *gin.Context) {
	var id int
	userId := b.getUserId(c)
	if userId == "" {
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	// If post exists: update, else create. Check if user is owner first.
	var newPost post
	if err := c.MustBindWith(&newPost, binding.FormPost); err != nil {
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}

	if newPost.Id != 0 {
		var owner string
		err := b.database.QueryRow(context.Background(), "SELECT owner FROM posts WHERE id=$1", newPost.Id).Scan(&owner)
		if isDBError(c, err) {
			return
		}
		if owner != userId {
			_ = c.AbortWithError(http.StatusForbidden, err)
			return
		}
		_, err = b.database.Exec(context.Background(), "UPDATE posts SET title=$1, content=$2 WHERE id=$3 RETURNING id", newPost.Title, newPost.Content, newPost.Id)
		if isDBError(c, err) {
			return
		}
		id = newPost.Id
	} else {
		err := b.database.QueryRow(context.Background(), "INSERT INTO posts (title, content, owner) VALUES ($1,$2, $3) RETURNING id", newPost.Title, newPost.Content, userId).Scan(&id)
		if isDBError(c, err) {
			return
		}
	}
	c.Redirect(http.StatusFound, b.config.PostPath+"/"+strconv.Itoa(id))
}

func (b *Controller) deletePost(c *gin.Context) {
	_, err := b.database.Exec(context.Background(), "DELETE FROM posts WHERE id=$1", c.Param("id"))
	if isDBError(c, err) {
		return
	}
	c.Status(http.StatusNoContent)
}

func (b *Controller) getFeed(c *gin.Context) {
	rows, err := b.database.Query(context.Background(), "SELECT id, title, content, publication_date, owner FROM posts ORDER BY id DESC LIMIT 10")
	if isDBError(c, err) {
		return
	}
	defer rows.Close()

	feed := make([]post, 0)
	for rows.Next() {
		p := post{}
		err := rows.Scan(&p.Id, &p.Title, &p.Content, &p.PublicationDate, &p.Owner)
		if isDBError(c, err) {
			return
		}
		feed = append(feed, p)
	}

	c.HTML(http.StatusOK, articleTemplate, gin.H{
		"user": b.getUserId(c),
		"feed": feed,
	})
}

func (b *Controller) adminArea(c *gin.Context) {
	c.HTML(http.StatusOK, adminTemplate, gin.H{
		"user": b.getUserId(c),
	})
}

func (b *Controller) getUserId(c *gin.Context) string {
	userSession := sessions.Default(c)
	userObject := userSession.Get("user")
	var userId string
	if userObject != nil {
		userId = userObject.(controller.UserObject).Id
	}
	return userId
}

func isDBError(c *gin.Context, err error) bool {
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			_ = c.AbortWithError(http.StatusNotFound, err)
			return true
		}
		_ = c.AbortWithError(http.StatusInternalServerError, err)
		return true
	}
	return false
}
