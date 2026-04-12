package engine

import (
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"
)

// HandleScan is the Gin HTTP handler for POST /scan
func HandleScan(c *gin.Context) {
	var req ScanRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "manifest and source fields required"})
		return
	}
	if req.Manifest == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "manifest must not be empty"})
		return
	}

	result, err := Run(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, result)
}

// parseJSON is a small helper used by scanner sub-checks
func parseJSON(data []byte, v any) error {
	return json.Unmarshal(data, v)
}
