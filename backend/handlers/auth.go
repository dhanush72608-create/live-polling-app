package handlers

import (
	"context"
	"net/http"
	"os"
	"time"

	"live-polling-backend/models"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"golang.org/x/crypto/bcrypt"
)

// Register creates a new user account
func Register(db *mongo.Database) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req models.AuthRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Valid email and minimum 6-character password required"})
			return
		}

		usersColl := db.Collection("users")
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		var existing models.User
		if err := usersColl.FindOne(ctx, bson.M{"email": req.Email}).Decode(&existing); err == nil {
			c.JSON(http.StatusConflict, gin.H{"error": "Email is already registered"})
			return
		}

		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to encrypt password"})
			return
		}

		newUser := models.User{
			Email:    req.Email,
			Password: string(hashedPassword),
		}

		res, err := usersColl.InsertOne(ctx, newUser)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error saving user"})
			return
		}

		c.JSON(http.StatusCreated, gin.H{"message": "User registered successfully", "id": res.InsertedID})
	}
}

// Login verifies credentials and issues a JWT
func Login(db *mongo.Database) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req models.AuthRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Email and password required"})
			return
		}

		usersColl := db.Collection("users")
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		var user models.User
		if err := usersColl.FindOne(ctx, bson.M{"email": req.Email}).Decode(&user); err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid email or password"})
			return
		}

		if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid email or password"})
			return
		}

		secret := os.Getenv("JWT_SECRET")
		if secret == "" {
			secret = "live_polling_secret_jwt_key_2026"
		}

		token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
			"user_id": user.ID.Hex(),
			"email":   user.Email,
			"exp":     time.Now().Add(72 * time.Hour).Unix(),
		})

		tokenString, err := token.SignedString([]byte(secret))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not generate authentication token"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"token": tokenString, "email": user.Email})
	}
}