package structures

import "go.mongodb.org/mongo-driver/bson/primitive"

type User struct {
	ID primitive.ObjectID `json:"id" bson:"_id,omitempty"`

	Twitch struct {
		ID             string `json:"id" bson:"id"`
		Login          string `json:"login" bson:"login"`
		DisplayName    string `json:"display_name" bson:"display_name"`
		ProfilePicture string `json:"profile_picture" bson:"profile_picture"`
	} `json:"twitch" bson:"twitch"`

	StreamKey string `json:"stream_key" bson:"stream_key"`
}
