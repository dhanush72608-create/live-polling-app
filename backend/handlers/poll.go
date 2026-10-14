package handlers

import (
	"context"
	"net/http"
	"time"

	"live-polling-backend/models"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

// CreatePoll validates input, writes poll metadata to MongoDB, and initializes Redis counts
func CreatePoll(db *mongo.Database, rdb *redis.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := c.Get("userID")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			return
		}

		var req models.CreatePollRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Question and at least 2 non-empty options required"})
			return
		}

		var options []models.Option
		for _, text := range req.Options {
			options = append(options, models.Option{
				ID:    uuid.New().String()[:8],
				Text:  text,
				Votes: 0,
			})
		}

		poll := models.Poll{
			CreatorID: userID.(primitive.ObjectID),
			Question:  req.Question,
			Options:   options,
			CreatedAt: time.Now().Unix(),
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		res, err := db.Collection("polls").InsertOne(ctx, poll)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save poll"})
			return
		}

		pollIDHex := res.InsertedID.(primitive.ObjectID).Hex()

		// Initialize zero-counts in Redis hash
		initMap := make(map[string]interface{})
		for _, opt := range options {
			initMap[opt.ID] = 0
		}
		rdb.HSet(ctx, "poll_counts:"+pollIDHex, initMap)

		c.JSON(http.StatusCreated, gin.H{
			"message": "Poll created successfully",
			"poll_id": pollIDHex,
		})
	}
}

// GetPoll returns poll metadata from MongoDB and live counts from Redis
func GetPoll(db *mongo.Database, rdb *redis.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		idHex := c.Param("id")
		objID, err := primitive.ObjectIDFromHex(idHex)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid poll ID format"})
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		var poll models.Poll
		if err := db.Collection("polls").FindOne(ctx, bson.M{"_id": objID}).Decode(&poll); err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Poll not found"})
			return
		}

		// Retrieve counts directly from Redis
		counts, err := rdb.HGetAll(ctx, "poll_counts:"+idHex).Result()
		if err != nil || len(counts) == 0 {
			counts = make(map[string]string)
			for _, opt := range poll.Options {
				counts[opt.ID] = "0"
			}
		}

		c.JSON(http.StatusOK, gin.H{
			"poll":   poll,
			"counts": counts,
		})
	}
}

// CastVote handles vote submissions: updates Redis counter, publishes live event, and asynchronously updates MongoDB
func CastVote(db *mongo.Database, rdb *redis.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		pollIDHex := c.Param("id")
		var req models.VoteRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Option ID required"})
			return
		}

		ctx := context.Background()

		// 1. Real-time update in Redis and Pub/Sub notification
		if err := BroadcastVote(ctx, rdb, pollIDHex, req.OptionID); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Real-time update failed"})
			return
		}

		// 2. Asynchronous MongoDB update for persistence
		go func(pID, oID string) {
			objID, err := primitive.ObjectIDFromHex(pID)
			if err != nil {
				return
			}
			filter := bson.M{"_id": objID, "options.id": oID}
			update := bson.M{"$inc": bson.M{"options.$.votes": 1}}
			_, _ = db.Collection("polls").UpdateOne(context.Background(), filter, update)
		}(pollIDHex, req.OptionID)

		c.JSON(http.StatusOK, gin.H{"message": "Vote recorded"})
	}
}