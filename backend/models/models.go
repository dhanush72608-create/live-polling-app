package models

import "go.mongodb.org/mongo-driver/bson/primitive"

// User represents an authenticated account
type User struct {
	ID       primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Email    string             `bson:"email" json:"email"`
	Password string             `bson:"password" json:"-"`
}

// AuthRequest handles login and registration payloads
type AuthRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=6"`
}

// Option represents a choice inside a poll
type Option struct {
	ID    string `bson:"id" json:"id"`
	Text  string `bson:"text" json:"text"`
	Votes int64  `bson:"votes" json:"votes"`
}

// Poll represents the poll document in MongoDB
type Poll struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	CreatorID primitive.ObjectID `bson:"creator_id" json:"creator_id"`
	Question  string             `bson:"question" json:"question"`
	Options   []Option           `bson:"options" json:"options"`
	CreatedAt int64              `bson:"created_at" json:"created_at"`
}

// CreatePollRequest validates incoming poll creation data
type CreatePollRequest struct {
	Question string   `json:"question" binding:"required,min=5"`
	Options  []string `json:"options" binding:"required,min=2,dive,required"`
}

// VoteRequest validates the vote submitted by an audience member
type VoteRequest struct {
	OptionID string `json:"option_id" binding:"required"`
}