package dataaccess

import (
	"errors"

	"go.mongodb.org/mongo-driver/v2/bson"
	"golang.org/x/crypto/bcrypt"

	"ships3d/models"
)

var (
	ErrUserExists         = errors.New("a user with that email or username already exists")
	ErrInvalidCredentials = errors.New("invalid email or password")
)

func (s *Store) GetUserByID(id bson.ObjectID) (*models.User, error) {
	return s.findUser(bson.D{{Key: "_id", Value: id}})
}

func (s *Store) findUser(filter bson.D) (*models.User, error) {
	ctx, cancel := newContext()
	defer cancel()
	var user models.User
	if err := s.collection(usersCollection).FindOne(ctx, filter).Decode(&user); err != nil {
		return nil, notFound(err)
	}
	return &user, nil
}

// Authenticate returns the user whose email and password match. Both failure
// cases return the same error, so a login form can't be used to find out which
// emails are registered.
func (s *Store) Authenticate(email, password string) (*models.User, error) {
	user, err := s.findUser(bson.D{{Key: "email", Value: email}})
	if errors.Is(err, ErrNotFound) {
		return nil, ErrInvalidCredentials
	}
	if err != nil {
		return nil, err
	}
	if bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)) != nil {
		return nil, ErrInvalidCredentials
	}
	return user, nil
}

// CreateUser registers a new user, refusing a duplicate email or username.
// The check-then-insert is not atomic (there is no unique index, since the
// collection is shared with ships-go and may already hold duplicates), so two
// simultaneous registrations of the same name could both succeed.
func (s *Store) CreateUser(username, email, password string) (*models.User, error) {
	_, err := s.findUser(bson.D{{Key: "$or", Value: bson.A{
		bson.D{{Key: "email", Value: email}},
		bson.D{{Key: "username", Value: username}},
	}}})
	if err == nil {
		return nil, ErrUserExists
	}
	if !errors.Is(err, ErrNotFound) {
		return nil, err
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}
	user := models.User{Username: username, Email: email, Password: string(hash)}

	ctx, cancel := newContext()
	defer cancel()
	res, err := s.collection(usersCollection).InsertOne(ctx, user)
	if err != nil {
		return nil, err
	}
	user.Id, _ = res.InsertedID.(bson.ObjectID)
	return &user, nil
}

// ChangePassword replaces a user's password after checking the current one.
func (s *Store) ChangePassword(id bson.ObjectID, current, next string) error {
	user, err := s.GetUserByID(id)
	if err != nil {
		return err
	}
	if bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(current)) != nil {
		return ErrInvalidCredentials
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(next), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	ctx, cancel := newContext()
	defer cancel()
	_, err = s.collection(usersCollection).UpdateByID(ctx, id, bson.D{{Key: "$set", Value: bson.D{
		{Key: "password", Value: string(hash)},
	}}})
	return err
}
