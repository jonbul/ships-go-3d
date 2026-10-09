package models

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// Session is a document of the `sessions` collection, shared with the 2D game
// (ships-go): logging in to either game logs in to both, since both read the
// same `token` cookie on the same host. Keep the bson tags identical to
// ships-go's models.Session.
type Session struct {
	Id               bson.ObjectID `json:"_id" bson:"_id,omitempty"`
	Admin            bool          `json:"admin" bson:"admin"`
	UserId           string        `json:"userId" bson:"userId"`
	SessionTimeStamp int64         `json:"sessionTimestamp" bson:"sessionTimestamp"`
	Persistent       bool          `json:"persistent" bson:"persistent"`
	Token            string        `json:"token" bson:"token"`
	LoggedOut        bool          `json:"loggedOut" bson:"loggedOut"`
	ExpirationTime   int64         `json:"expirationTime" bson:"expirationTime"`
}

func (s *Session) UserIdAsBsonObject() bson.ObjectID {
	id, _ := bson.ObjectIDFromHex(s.UserId)
	return id
}

// IsValid reports whether the session can still authenticate a request.
// ships-go never checks LoggedOut, so a logged-out token stays usable there
// until it expires; here it does not.
func (s *Session) IsValid(now time.Time) bool {
	if s.LoggedOut {
		return false
	}
	if s.Persistent {
		return true
	}
	return s.ExpirationTime > now.UnixMilli()
}
