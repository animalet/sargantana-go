package blog

import (
	"errors"
	"strconv"

	"github.com/animalet/sargantana-go/config"
	"github.com/animalet/sargantana-go/controller"
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx"
)

type (
	Controller struct {
		config   *Config
		database *pgx.Conn
	}
	Config struct {
		FeedUrl      string `yaml:"feed_url"`
		PostUrl      string `yaml:"post_url"`
		AdminAreaUrl string `yaml:"admin_area_url"`
	}
)

func (b Config) Validate() error {
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

func (b *Controller) Bind(engine *gin.Engine, loginMiddleware gin.HandlerFunc) {
	api := engine.Group("/api")
	{
		api.GET(b.config.FeedUrl, b.getFeed)
		api.POST(b.config.PostUrl, b.createPost)
		api.GET(b.config.PostUrl+"/:id", b.getPost)
		api.DELETE(b.config.PostUrl+"/:id", b.deletePost)
		api.GET(b.config.AdminAreaUrl, loginMiddleware, b.adminArea)
	}
}

func (b *Controller) Close() error { return nil }

func NewBlogController(db *pgx.Conn) controller.Constructor {
	return func(configData config.ControllerConfig, _ config.ServerConfig) (controller.IController, error) {
		_, err := db.Exec(`CREATE TABLE IF NOT EXISTS posts (
        id SERIAL PRIMARY KEY,
        title TEXT NOT NULL,
        content TEXT NOT NULL,
        publication_date TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
        owner TEXT NOT NULL
    );`)

		if err != nil {
			return nil, err
		}

		cfg, err := config.UnmarshalTo[Config](configData)
		if err != nil {
			return nil, err
		}
		return &Controller{config: cfg, database: db}, nil
	}
}

type post struct {
	id              int
	title           string
	content         string
	publicationDate string
	owner           string
}

const (
	articleTemplate = "articles.html"
	adminTemplate   = "admin.html"
)

func (b *Controller) getPost(c *gin.Context) {
	p := post{}
	err := b.database.QueryRow("SELECT id, title, content, publication_date, owner FROM posts WHERE id=$1", c.Param("id")).
		Scan(&p.id, &p.title, &p.content, &p.publicationDate, &p.owner)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(404, gin.H{"error": "Post not found"})
			return
		}
		c.JSON(500, gin.H{"error": "Database error"})
	}

	c.HTML(200, articleTemplate, gin.H{
		"feed": []post{p},
	})
}

func (b *Controller) createPost(c *gin.Context) {
	var id int
	userId := b.getUserId(c)
	if userId == "" {
		c.JSON(401, gin.H{"error": "Unauthorized"})
		return
	}

	// If post exists, update, else, create. Check if user is owner first.
	var newPost post
	if err := c.BindJSON(&newPost); err != nil {
		c.JSON(400, gin.H{"error": "Invalid request"})
		return
	}

	if newPost.id != 0 {
		var owner string
		err := b.database.QueryRow("SELECT owner FROM posts WHERE id=$1", newPost.id).Scan(&owner)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				c.JSON(404, gin.H{"error": "Post not found"})
				return
			}
			c.JSON(500, gin.H{"error": "Database error"})
			return
		}
		if owner != userId {
			c.JSON(403, gin.H{"error": "Forbidden"})
			return
		}
		_, err = b.database.Exec("UPDATE posts SET title=$1, content=$2 WHERE id=$3 RETURNING id", newPost.title, newPost.content, newPost.id)
		if err != nil {
			c.JSON(500, gin.H{"error": "Database error"})
			return
		}
		id = newPost.id
	} else {
		err := b.database.QueryRow("INSERT INTO posts (title, content, owner) VALUES ($1,$2, $3) RETURNING id", newPost.title, newPost.content, userId).Scan(&id)
		if err != nil {
			c.JSON(500, gin.H{"error": "Database error"})
			return
		}
	}
	c.Redirect(302, b.config.PostUrl+"/"+strconv.Itoa(id))
}

func (b *Controller) deletePost(c *gin.Context) {
	err := b.database.QueryRow("DELETE FROM posts WHERE id=$1", c.Param("id"))
	if err != nil {
		c.JSON(500, gin.H{"error": "Database error"})
		return
	}
	c.Redirect(302, b.config.FeedUrl)
}

func (b *Controller) getFeed(c *gin.Context) {
	rows, err := b.database.Query("SELECT id, title, content, publication_date, owner FROM posts ORDER BY id DESC LIMIT 10")
	if err != nil {
		c.JSON(500, gin.H{"error": "Database error"})
		return
	}
	defer rows.Close()

	feed := make([]post, 0)
	for rows.Next() {
		p := post{}
		err := rows.Scan(&p.id, &p.title, &p.content, &p.publicationDate, &p.owner)

		if err != nil {
			c.JSON(500, gin.H{"error": "Database error"})
			return
		}
		feed = append(feed, p)
	}

	c.HTML(200, articleTemplate, gin.H{
		"user": b.getUserId(c),
		"feed": feed,
	})
}

func (b *Controller) adminArea(c *gin.Context) {
	c.HTML(200, adminTemplate, nil)
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
