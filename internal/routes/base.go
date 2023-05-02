package routes

import (
	"encoding/base64"
	"i3_stat/internal/entities"
	"i3_stat/internal/logger"
	"i3_stat/internal/service/sampler"
	"i3_stat/internal/service/telegramist"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"go.uber.org/zap"
)

type Server struct {
	appAddr    string
	log        logger.AppLogger
	service    *sampler.Service
	sender     *telegramist.Sender
	httpEngine *fiber.App
}

// InitAppRouter initializes the HTTP Server.
func InitAppRouter(log logger.AppLogger, service *sampler.Service, sender *telegramist.Sender, address string) *Server {
	app := &Server{
		appAddr:    address,
		httpEngine: fiber.New(fiber.Config{}),
		service:    service,
		sender:     sender,
		log:        log.With(zap.String("service", "http")),
	}
	app.httpEngine.Use(recover.New())
	app.initRoutes()
	return app
}

func (s *Server) initRoutes() {
	s.httpEngine.Get("/", func(ctx *fiber.Ctx) error {
		return ctx.SendString("pong")
	})
	s.httpEngine.Get("/state", func(ctx *fiber.Ctx) error {
		return ctx.SendString(s.service.GetState())
	})

	s.httpEngine.Post("/instant_message", func(c *fiber.Ctx) error {
		decodedBytes, err := base64.StdEncoding.DecodeString(string(c.Body()))
		if err != nil {
			return c.SendStatus(fiber.StatusBadRequest)
		}
		s.sender.InstantMessage(string(decodedBytes))
		return c.SendString("ok")
	})
	s.httpEngine.Post("/message", func(c *fiber.Ctx) error {
		decodedBytes, err := base64.StdEncoding.DecodeString(string(c.Body()))
		if err != nil {
			return c.SendStatus(fiber.StatusBadRequest)
		}
		s.sender.HandleMessage(entities.Message{
			Messages: []string{string(decodedBytes)},
			Time:     time.Now(),
		})
		return c.SendString("ok")
	})
}

// Run starts the HTTP Server.
func (s *Server) Run() error {
	s.log.Info("Starting HTTP server", zap.String("port", s.appAddr))
	return s.httpEngine.Listen(s.appAddr)
}

func (s *Server) Stop() error {
	return s.httpEngine.Shutdown()
}
