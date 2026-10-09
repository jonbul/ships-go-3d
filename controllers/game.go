package controllers

import (
	"log"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/v2/bson"

	"ships3d/game"
	"ships3d/models"
)

func (s *Server) registerGameRoutes(router *gin.Engine) {
	router.GET("/game/ships", s.gameShips)
	router.GET("/ws", func(c *gin.Context) {
		s.hub.ServeWS(c.Writer, c.Request)
	})
}

// gameShips lists the ships a player can choose from: the built-in ones, and
// their own projects when logged in.
func (s *Server) gameShips(c *gin.Context) {
	own := []game.ShipInfo{}
	if user, _ := s.currentUser(c); user != nil {
		projects, err := s.store.GetProjectsByUserId(user.IdAsString())
		if err != nil {
			log.Println("gameShips:", err)
		}
		for _, project := range projects {
			if project.BoundingRadius() > 0 {
				own = append(own, game.ShipInfo{Id: project.Id.Hex(), Name: project.Name, Layers: project.Layers})
			}
		}
	}
	c.JSON(http.StatusOK, gin.H{"defaults": game.DefaultShips(), "own": own})
}

// identify resolves the player behind a websocket handshake from its session
// cookie. No valid session means a guest.
func (s *Server) identify(r *http.Request) game.Identity {
	cookie, err := r.Cookie(tokenCookie)
	if err != nil {
		return game.Identity{}
	}
	session := s.sessionFromToken(cookie.Value)
	if session == nil {
		return game.Identity{}
	}
	user, err := s.store.GetUserByID(session.UserIdAsBsonObject())
	if err != nil {
		return game.Identity{}
	}
	return game.Identity{UserId: user.IdAsString(), Username: user.Username}
}

// loadShip returns a built-in ship to anyone, and a project only to its owner.
func (s *Server) loadShip(identity game.Identity, shipId string) (*models.Project3d, error) {
	if strings.HasPrefix(shipId, game.DefaultShipPrefix) {
		if ship, ok := game.DefaultShip(shipId); ok {
			return ship, nil
		}
		return nil, game.ErrShipNotAllowed
	}
	if identity.Guest() {
		return nil, game.ErrShipNotAllowed
	}
	id, err := bson.ObjectIDFromHex(shipId)
	if err != nil {
		return nil, game.ErrShipNotAllowed
	}
	return s.store.GetUserProject(id, identity.UserId)
}
