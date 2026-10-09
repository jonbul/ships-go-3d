package models

import "go.mongodb.org/mongo-driver/v2/bson"

// User is a document of the `users` collection, which is shared with the 2D
// game (ships-go). Field names and bson tags must stay identical to ships-go's
// models.User, or a user registered in one game breaks in the other.
type User struct {
	Id       bson.ObjectID `json:"_id" bson:"_id,omitempty"`
	Admin    bool          `json:"admin" bson:"admin"`
	Username string        `json:"username" bson:"username"`
	// Password holds the bcrypt hash. `json:"-"` keeps it out of every API
	// response, whatever handler happens to serialise a User.
	Password string `json:"-" bson:"password"`
	Email    string `json:"email" bson:"email"`
	Credits  int    `json:"credits" bson:"credits"`
	Kills    int    `json:"kills" bson:"kills"`
	Deaths   int    `json:"deaths" bson:"deaths"`
}

func (u *User) IdAsString() string {
	return u.Id.Hex()
}
