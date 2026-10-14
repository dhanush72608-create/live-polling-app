package main

import (
	"context"
	"log"
	"os"
	"time"

	"live-polling-backend/handlers"
	"live-polling-backend/middleware"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func main() {
	mongoURI := os.Getenv("MONGO_URI")
	if mongoURI == "" {
		log.Println("WARNING: MONGO_URI is not set. Defaulting to localhost:27017")
		mongoURI = "mongodb://localhost:27017"
	}

	redisURL := os.Getenv("REDIS_URL")
	if redisURL == "" {
		log.Println("WARNING: REDIS_URL is not set. Defaulting to redis://localhost:6379")
		redisURL = "redis://localhost:6379"
	}

	// 1. Connect to MongoDB
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	mongoClient, err := mongo.Connect(ctx, options.Client().ApplyURI(mongoURI))
	if err != nil {
		log.Fatalf("MongoDB connection error: %v", err)
	}
	db := mongoClient.Database("livepoll")
	log.Println("Connected to MongoDB")

	// 2. Connect to Redis
	redisOpt, err := redis.ParseURL(redisURL)
	if err != nil {
		log.Fatalf("Redis URL parsing error: %v", err)
	}
	rdb := redis.NewClient(redisOpt)
	if err := rdb.Ping(context.Background()).Err(); err != nil {
		log.Printf("Warning: Redis ping issue: %v", err)
	} else {
		log.Println("Connected to Redis successfully")
	}

	// 3. Setup Gin router with CORS
	r := gin.Default()

	r.Use(cors.New(cors.Config{
		AllowAllOrigins:  true,
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	// Health check endpoint
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	// Public API routes
	api := r.Group("/api")
	{
		api.POST("/auth/register", handlers.Register(db))
		api.POST("/auth/login", handlers.Login(db))
		api.GET("/polls/:id", handlers.GetPoll(db, rdb))
		api.POST("/polls/:id/vote", handlers.CastVote(db, rdb))
	}

	// Protected API routes (requires JWT)
	protected := api.Group("/polls")
	protected.Use(middleware.RequireAuth())
	{
		protected.POST("", handlers.CreatePoll(db, rdb))
	}

	// Real-time WebSocket route
	r.GET("/ws/polls/:id", handlers.HandleWebSocket(rdb))

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("Server listening on port %s", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatalf("Failed to run server: %v", err)
	}
}