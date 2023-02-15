package main

import (
	"encoding/json"
	"fmt"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"golang.org/x/net/context"
	"net/http"
	"posts/configs"
	"time"
)

type Post struct {
	ID          primitive.ObjectID `json:"_id,omitempty" bson:"_id,omitempty"`
	Title       string             `json:"title,omitempty" bson:"title,omitempty"`
	Description string             `json:"description,omitempty" bson:"description,omitempty"`
	Comment     []Comment          `json:"comment,omitempty" bson:"comment,omitempty" default:"[]"`
}
type Comment struct {
	ID     primitive.ObjectID `json:"_id,omitempty" bson:"_id,omitempty"`
	PostId string             `json:"postId,omitempty" bson:"postId,omitempty"`
	Text   string             `json:"text,omitempty" bson:"text,omitempty"`
}

func main() {

	config, err := configs.LoadConfig(".")
	if err != nil {
		panic("Error loading config file")
	}

	ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
	client, _ := mongo.Connect(ctx, options.Client().ApplyURI(config.MongoURI))
	db := client.Database("go_search")

	app := fiber.New()

	app.Use(cors.New())

	app.Get("/api/posts", func(c *fiber.Ctx) error {
		collection := db.Collection("posts")
		ctx, _ := context.WithTimeout(context.Background(), 30*time.Second)
		var posts []Post

		cursor, err := collection.Find(ctx, Post{})
		if err != nil {
			return c.Status(500).SendString(err.Error())
		}
		if err = cursor.All(ctx, &posts); err != nil {
			return c.Status(500).SendString(err.Error())
		}
		for i, post := range posts {
			res, err := http.Get(fmt.Sprintf("http://localhost:8001/api/posts/%s/comments", post.ID.Hex()))

			if err != nil {
				return c.Status(500).SendString(err.Error())
			}
			var comments []Comment
			if err := json.NewDecoder(res.Body).Decode(&comments); err != nil {
				return c.Status(500).SendString(err.Error())
			}
			posts[i].Comment = comments
		}
		return c.JSON(fiber.Map{"posts": posts})
	})

	app.Get("/api/posts/:id", func(c *fiber.Ctx) error {
		collection := db.Collection("posts")
		ctx, _ := context.WithTimeout(context.Background(), 30*time.Second)
		var post Post
		id := c.Params("id")
		objID, _ := primitive.ObjectIDFromHex(id)
		err := collection.FindOne(ctx, Post{ID: objID}).Decode(&post)
		if err != nil {
			return c.Status(500).SendString(err.Error())
		}
		res, err := http.Get(fmt.Sprintf("http://localhost:8001/api/posts/%s/comments", id))
		if err != nil {
			return c.Status(500).SendString(err.Error())
		}
		var comments []Comment
		if err := json.NewDecoder(res.Body).Decode(&comments); err != nil {
			return c.Status(500).SendString(err.Error())
		}
		post.Comment = comments
		return c.JSON(fiber.Map{"post": post})
	})

	app.Post("/api/posts", func(c *fiber.Ctx) error {
		collection := db.Collection("posts")
		ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)

		post := new(Post)
		if err := c.BodyParser(post); err != nil {
			return c.Status(500).SendString(err.Error())
		}
		post.ID = primitive.NewObjectID()
		_, err := collection.InsertOne(ctx, post)
		if err != nil {
			return c.Status(500).SendString(err.Error())
		}
		return c.JSON(fiber.Map{"post": post})

	})

	app.Listen(":" + config.ServerPort)
}
