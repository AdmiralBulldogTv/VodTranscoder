package structures

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Vod struct {
	ID     primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	UserID primitive.ObjectID `json:"user_id" bson:"user_id"`

	Title string `json:"title" bson:"title"`

	Categories []VodCategory `json:"categories" bson:"categories"`

	State      VodState      `json:"vod_state" bson:"vod_state"`
	Visibility VodVisibility `json:"vod_visibility" bson:"vod_visibility"`

	Variants []VodVariant `json:"variants" bson:"variants"`

	StartedAt time.Time `json:"started_at" bson:"started_at"`
	EndedAt   time.Time `json:"ended_at" bson:"ended_at"`
}

type VodCategory struct {
	Timestamp time.Time `json:"timestamp" bson:"timestamp"`
	Name      string    `json:"name" bson:"name"`
	ID        string    `json:"id" bson:"id"`
	URL       string    `json:"url" bson:"url"`
}

type VodState int32

const (
	VodStateLive VodState = iota
	VodStateQueued
	VodStateProcessing
	VodStateReady
	VodStateStorage
	VodStateFailed
	VodStateCanceled
)

type VodVisibility int32

const (
	VodVisibilityPublic VodVisibility = iota
	VodVisibilityHidden
	VodVisibilityDeleted
)

type VodVariant struct {
	Name    string `json:"name" bson:"name"`
	Width   int    `json:"width" bson:"width"`
	Height  int    `json:"height" bson:"height"`
	FPS     int    `json:"fps" bson:"fps"`
	Bitrate int    `json:"bitrate" bson:"bitrate"`
	Ready   bool   `json:"ready" bson:"ready"`
}
