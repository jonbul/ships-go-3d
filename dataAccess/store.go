package dataaccess

import (
	"context"
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// Collection names. `users` and `sessions` are shared with the 2D game
// (ships-go); 3D painting projects get a collection of their own because
// their documents have a different shape from the 2D `paintingprojects`.
const (
	usersCollection    = "users"
	sessionsCollection = "sessions"
	projectsCollection = "paintingProjects3d"
)

// queryTimeout bounds every database call, so a stalled MongoDB fails a
// request instead of hanging it forever.
const queryTimeout = 10 * time.Second

// ErrNotFound is returned when a looked-up document does not exist.
var ErrNotFound = errors.New("not found")

// Store holds one MongoDB client for the life of the process. The driver
// pools connections itself, so connecting per query (as ships-go does) only
// adds a handshake to every request.
type Store struct {
	client *mongo.Client
	db     *mongo.Database
}

func Connect(uri, database string) (*Store, error) {
	client, err := mongo.Connect(options.Client().ApplyURI(uri))
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), queryTimeout)
	defer cancel()
	if err := client.Ping(ctx, nil); err != nil {
		_ = client.Disconnect(context.Background())
		return nil, err
	}
	return &Store{client: client, db: client.Database(database)}, nil
}

func (s *Store) Close() error {
	ctx, cancel := context.WithTimeout(context.Background(), queryTimeout)
	defer cancel()
	return s.client.Disconnect(ctx)
}

func (s *Store) collection(name string) *mongo.Collection {
	return s.db.Collection(name)
}

func newContext() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), queryTimeout)
}

// notFound maps the driver's "no documents" error to ErrNotFound, so callers
// don't need to import the driver to tell "missing" from "failed".
func notFound(err error) error {
	if errors.Is(err, mongo.ErrNoDocuments) {
		return ErrNotFound
	}
	return err
}
