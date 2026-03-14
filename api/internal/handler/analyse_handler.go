package handler

import (
	"log"
	"net/http"

	"contract-scanner/internal/middleware"
	"contract-scanner/internal/usecase"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type AnalyseHandler struct {
	processContract usecase.IProcessContract
	listAnalyses    usecase.IListAnalyses
	deleteAnalysis  usecase.IDeleteAnalysis
}

func NewAnalyseHandler(processContract usecase.IProcessContract, listAnalyses usecase.IListAnalyses, deleteAnalysis usecase.IDeleteAnalysis) *AnalyseHandler {
	return &AnalyseHandler{
		processContract: processContract,
		listAnalyses:    listAnalyses,
		deleteAnalysis:  deleteAnalysis,
	}
}

func (h *AnalyseHandler) List(c *gin.Context) {
	userID := middleware.GetUserID(c)
	items, err := h.listAnalyses.Execute(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"analyses": items})
}

func (h *AnalyseHandler) Process(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid analyse id"})
		return
	}

	log.Printf("[process] starting analyse_id=%s", id)

	output, err := h.processContract.Execute(c.Request.Context(), usecase.ProcessInput{
		AnalyseID:   id,
		ClerkUserID: middleware.GetUserID(c),
	})
	if err != nil {
		log.Printf("[process] failed analyse_id=%s error=%v", id, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	log.Printf("[process] completed analyse_id=%s status=%s", id, output.Status)
	c.JSON(http.StatusOK, output)
}

func (h *AnalyseHandler) Delete(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid analyse id"})
		return
	}
	userID := middleware.GetUserID(c)
	if err := h.deleteAnalysis.Execute(c.Request.Context(), id, userID); err != nil {
		if err.Error() == "not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "analysis not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}
