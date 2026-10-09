package dataaccess

import (
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"ships3d/models"
)

func (s *Store) GetProjectsByUserId(userId string) ([]models.Project3d, error) {
	ctx, cancel := newContext()
	defer cancel()
	cursor, err := s.collection(projectsCollection).Find(ctx,
		bson.D{{Key: "userId", Value: userId}},
		options.Find().SetSort(bson.D{{Key: "dateModified", Value: -1}}),
	)
	if err != nil {
		return nil, err
	}
	projects := []models.Project3d{}
	err = cursor.All(ctx, &projects)
	return projects, err
}

// GetUserProject returns a project only if it belongs to the given user. Every
// lookup goes through the owner, so one user can never read, overwrite or
// delete another's project by guessing its id.
func (s *Store) GetUserProject(id bson.ObjectID, userId string) (*models.Project3d, error) {
	ctx, cancel := newContext()
	defer cancel()
	var project models.Project3d
	err := s.collection(projectsCollection).FindOne(ctx, ownedBy(id, userId)).Decode(&project)
	if err != nil {
		return nil, notFound(err)
	}
	return &project, nil
}

func (s *Store) InsertProject(project *models.Project3d) error {
	ctx, cancel := newContext()
	defer cancel()
	project.Id = bson.NilObjectID
	res, err := s.collection(projectsCollection).InsertOne(ctx, project)
	if err != nil {
		return err
	}
	project.Id, _ = res.InsertedID.(bson.ObjectID)
	return nil
}

// UpdateProject overwrites a project's content. DateCreated is kept as stored.
func (s *Store) UpdateProject(project *models.Project3d) error {
	ctx, cancel := newContext()
	defer cancel()
	res, err := s.collection(projectsCollection).UpdateOne(ctx, ownedBy(project.Id, project.UserId),
		bson.D{{Key: "$set", Value: bson.D{
			{Key: "name", Value: project.Name},
			{Key: "dateModified", Value: project.DateModified},
			{Key: "layers", Value: project.Layers},
		}}},
	)
	if err != nil {
		return err
	}
	if res.MatchedCount == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) DeleteUserProject(id bson.ObjectID, userId string) error {
	ctx, cancel := newContext()
	defer cancel()
	res, err := s.collection(projectsCollection).DeleteOne(ctx, ownedBy(id, userId))
	if err != nil {
		return err
	}
	if res.DeletedCount == 0 {
		return ErrNotFound
	}
	return nil
}

func ownedBy(id bson.ObjectID, userId string) bson.D {
	return bson.D{{Key: "_id", Value: id}, {Key: "userId", Value: userId}}
}
