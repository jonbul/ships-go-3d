package dataaccess

import (
	"go.mongodb.org/mongo-driver/v2/bson"

	"ships3d/models"
)

func (s *Store) InsertSession(session *models.Session) error {
	ctx, cancel := newContext()
	defer cancel()
	res, err := s.collection(sessionsCollection).InsertOne(ctx, session)
	if err != nil {
		return err
	}
	session.Id, _ = res.InsertedID.(bson.ObjectID)
	return nil
}

func (s *Store) GetSessionByToken(token string) (*models.Session, error) {
	ctx, cancel := newContext()
	defer cancel()
	var session models.Session
	err := s.collection(sessionsCollection).FindOne(ctx, bson.D{{Key: "token", Value: token}}).Decode(&session)
	if err != nil {
		return nil, notFound(err)
	}
	return &session, nil
}

func (s *Store) UpdateSession(session *models.Session) error {
	ctx, cancel := newContext()
	defer cancel()
	_, err := s.collection(sessionsCollection).UpdateByID(ctx, session.Id, bson.D{{Key: "$set", Value: session}})
	return err
}

// LogoutUserSessions ends every session of a user except the one given, so a
// password change signs out other devices.
func (s *Store) LogoutUserSessions(userId string, except bson.ObjectID) error {
	ctx, cancel := newContext()
	defer cancel()
	_, err := s.collection(sessionsCollection).UpdateMany(ctx,
		bson.D{{Key: "userId", Value: userId}, {Key: "_id", Value: bson.D{{Key: "$ne", Value: except}}}},
		bson.D{{Key: "$set", Value: bson.D{{Key: "loggedOut", Value: true}, {Key: "persistent", Value: false}}}},
	)
	return err
}
