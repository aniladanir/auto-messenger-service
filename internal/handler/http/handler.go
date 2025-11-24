package handler

import (
	"context"
	"net/http"

	_ "github.com/aniladanir/auto-messender-service/docs"
	"github.com/aniladanir/auto-messender-service/internal/service"
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

type Handler struct {
	msgSender service.MessageSender
	server    *http.Server
}

// @title Auto Messenger API
// @version 1.0
// @description API for automatic message sending service
// @host localhost:6060
// @BasePath /
func NewHttpHandler(addr string, svc service.MessageSender) *Handler {
	h := &Handler{
		msgSender: svc,
	}

	// create router
	router := gin.Default()

	// register routes
	router.POST("/start", h.startProcess)
	router.POST("/stop", h.stopProcess)
	router.GET("/messages", h.getSentMessages)
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// create http server
	h.server = &http.Server{
		Addr:    addr,
		Handler: router.Handler(),
	}

	return h
}

func (h *Handler) Run() error {
	return h.server.ListenAndServe()
}

func (h *Handler) Shutdown(ctx context.Context) error {
	return h.server.Shutdown(ctx)
}

// StartProcess godoc
// @Summary Start the automatic message sender
// @Description Starts the background process that sends x messages every y minutes
// @Tags Control
// @Success 200
// @Router /start [post]
func (h *Handler) startProcess(c *gin.Context) {
	h.msgSender.Start()
	c.Status(http.StatusOK)
}

// StopProcess godoc
// @Summary Stop the automatic message sender
// @Description Stops the background sending process
// @Tags Control
// @Success 200
// @Router /stop [post]
func (h *Handler) stopProcess(c *gin.Context) {
	h.msgSender.Stop()
	c.Status(http.StatusOK)
}

// GetSentMessages godoc
// @Summary Get list of sent messages
// @Description Retrieves all messages marked as sent
// @Tags Messages
// @Success 200 {array} domain.Message
// @Router /messages [get]
func (h *Handler) getSentMessages(c *gin.Context) {
	msgs, err := h.msgSender.GetSentMessages()
	if err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}
	c.JSON(http.StatusOK, msgs)
}
