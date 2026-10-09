package controllers

import (
	"errors"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/v2/bson"

	dataaccess "ships3d/dataAccess"
	"ships3d/models"
)

func (s *Server) registerProjectRoutes(router *gin.Engine) {
	projects := router.Group("/projects", s.requireUser)
	projects.GET("", s.listProjects)
	projects.POST("", s.createProject)
	projects.GET("/:id", s.getProject)
	projects.PUT("/:id", s.updateProject)
	projects.DELETE("/:id", s.deleteProject)
}

func (s *Server) listProjects(c *gin.Context) {
	projects, err := s.store.GetProjectsByUserId(contextUser(c).IdAsString())
	if err != nil {
		log.Println("listProjects:", err)
		errorResponse(c, http.StatusInternalServerError, "could not load the projects")
		return
	}
	c.JSON(http.StatusOK, projects)
}

func (s *Server) getProject(c *gin.Context) {
	id, ok := projectId(c)
	if !ok {
		return
	}
	project, err := s.store.GetUserProject(id, contextUser(c).IdAsString())
	if !projectFound(c, err) {
		return
	}
	c.JSON(http.StatusOK, project)
}

func (s *Server) createProject(c *gin.Context) {
	project, ok := bindProject(c)
	if !ok {
		return
	}
	now := time.Now().UnixMilli()
	project.UserId = contextUser(c).IdAsString()
	project.DateCreated = now
	project.DateModified = now
	if err := s.store.InsertProject(project); err != nil {
		log.Println("createProject:", err)
		errorResponse(c, http.StatusInternalServerError, "could not save the project")
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "project": project})
}

func (s *Server) updateProject(c *gin.Context) {
	id, ok := projectId(c)
	if !ok {
		return
	}
	project, ok := bindProject(c)
	if !ok {
		return
	}
	project.Id = id
	project.UserId = contextUser(c).IdAsString()
	project.DateModified = time.Now().UnixMilli()
	if !projectFound(c, s.store.UpdateProject(project)) {
		return
	}
	// Answer with the stored document: the request doesn't carry the fields
	// the update leaves alone, such as dateCreated.
	stored, err := s.store.GetUserProject(id, project.UserId)
	if !projectFound(c, err) {
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "project": stored})
}

func (s *Server) deleteProject(c *gin.Context) {
	id, ok := projectId(c)
	if !ok {
		return
	}
	if !projectFound(c, s.store.DeleteUserProject(id, contextUser(c).IdAsString())) {
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

func projectId(c *gin.Context) (bson.ObjectID, bool) {
	id, err := bson.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		errorResponse(c, http.StatusNotFound, "project not found")
		return id, false
	}
	return id, true
}

func bindProject(c *gin.Context) (*models.Project3d, bool) {
	var project models.Project3d
	if err := c.ShouldBindJSON(&project); err != nil {
		errorResponse(c, http.StatusBadRequest, "invalid project")
		return nil, false
	}
	if err := project.Validate(); err != nil {
		errorResponse(c, http.StatusBadRequest, err.Error())
		return nil, false
	}
	return &project, true
}

// projectFound turns a store error into a response. A project that exists but
// belongs to someone else is reported as not found, so ids can't be probed.
func projectFound(c *gin.Context, err error) bool {
	if errors.Is(err, dataaccess.ErrNotFound) {
		errorResponse(c, http.StatusNotFound, "project not found")
		return false
	}
	if err != nil {
		log.Println("project:", err)
		errorResponse(c, http.StatusInternalServerError, "could not access the project")
		return false
	}
	return true
}
