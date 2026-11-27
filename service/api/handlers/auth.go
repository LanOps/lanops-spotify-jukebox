package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func (s Client) AuthCallback(c *gin.Context) {
	if st := c.Request.FormValue("state"); !s.jbc.CheckState(st) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "state mismatch"})
		return
	}
	if spotifyErr := c.Request.FormValue("error"); spotifyErr != "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "spotify authorization failed: " + spotifyErr})
		return
	}
	token, err := s.jbc.GetAuthTokenFromRequest(c.Request.Context(), c.Request)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if err = s.jbc.Login(token); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Don't echo the OAuth token back to the browser
	c.JSON(http.StatusOK, gin.H{"message": "Spotify authorized"})
}

func (s Client) AuthLogin(c *gin.Context) {
	c.Redirect(302, s.jbc.GetAuthUrl())
}
