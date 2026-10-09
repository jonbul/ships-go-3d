package controllers

import (
	"crypto/rand"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"ships3d/models"
)

// The session cookie. Its name is shared with ships-go on purpose: cookies
// are scoped by host, not port, and both games read the same `sessions`
// collection, so logging in to either logs in to both.
const tokenCookie = "token"

const (
	// sessionDuration is how long a non-persistent session lives without
	// use. It slides: a session used after half of it has passed is extended.
	sessionDuration = 30 * 24 * time.Hour
	// persistentCookieAge is the cookie lifetime for "remember me".
	persistentCookieAge = 365 * 24 * time.Hour
)

const (
	ctxSession = "session"
	ctxUser    = "user"
)

func newToken() string {
	return rand.Text() + rand.Text()
}

func (s *Server) setTokenCookie(c *gin.Context, session *models.Session) {
	c.SetSameSite(http.SameSiteLaxMode)
	// A non-persistent session gets a browser-session cookie (no Max-Age),
	// gone when the browser closes, while the server still bounds it with
	// ExpirationTime.
	maxAge := 0
	if session.Persistent {
		maxAge = int(persistentCookieAge.Seconds())
	}
	c.SetCookie(tokenCookie, session.Token, maxAge, "/", "", true, true)
}

func clearTokenCookie(c *gin.Context) {
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(tokenCookie, "", -1, "/", "", true, true)
}

// sessionFromToken looks up a valid session. Unlike ships-go it never rotates
// the token on a normal request: rotating on every call made two requests
// sent at once race, with the loser presenting an already-replaced token and
// being logged out.
func (s *Server) sessionFromToken(token string) *models.Session {
	if token == "" {
		return nil
	}
	session, err := s.store.GetSessionByToken(token)
	if err != nil {
		return nil
	}
	now := time.Now()
	if !session.IsValid(now) {
		return nil
	}
	if !session.Persistent && time.UnixMilli(session.ExpirationTime).Sub(now) < sessionDuration/2 {
		session.ExpirationTime = now.Add(sessionDuration).UnixMilli()
		if err := s.store.UpdateSession(session); err != nil {
			log.Println("session: extend:", err)
		}
	}
	return session
}

// currentUser returns the logged-in user and their session, or nils.
func (s *Server) currentUser(c *gin.Context) (*models.User, *models.Session) {
	token, _ := c.Cookie(tokenCookie)
	session := s.sessionFromToken(token)
	if session == nil {
		return nil, nil
	}
	user, err := s.store.GetUserByID(session.UserIdAsBsonObject())
	if err != nil {
		return nil, nil
	}
	return user, session
}

// requireUser rejects requests without a valid session, and stores the user
// and session in the context for the handler.
func (s *Server) requireUser(c *gin.Context) {
	user, session := s.currentUser(c)
	if user == nil {
		errorResponse(c, http.StatusUnauthorized, "you need to log in")
		return
	}
	c.Set(ctxUser, user)
	c.Set(ctxSession, session)
	c.Next()
}

func contextUser(c *gin.Context) *models.User {
	return c.MustGet(ctxUser).(*models.User)
}

func contextSession(c *gin.Context) *models.Session {
	return c.MustGet(ctxSession).(*models.Session)
}
