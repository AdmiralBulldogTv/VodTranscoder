package structures

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Chat struct {
	ID     primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	Twitch struct {
		ID          string `json:"id" bson:"id"`
		Login       string `json:"login" bson:"login"`
		DisplayName string `json:"display_name" bson:"display_name"`
	} `json:"twitch" bson:"twitch"`
	Timestamp time.Time   `json:"timestamp" bson:"timestamp"`
	Content   string      `json:"content" bson:"content"`
	Badges    []ChatBadge `json:"badges" bson:"badges"`
	Emotes    []ChatEmote `json:"emotes" bson:"chat_emote"`
}

type ChatBadge struct {
	Name string `json:"name" bson:"name"`
	URL  string `json:"url" bson:"url"`
}

type ChatEmote struct {
	Name string `json:"name" bson:"name"`
	URL  string `json:"url" bson:"url"`
}
