package main

import (
	//"fmt"
	"time"
	"flag"
	"os"
	"os/signal"
	"context"
	"net/http"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	limiter "github.com/throttled/throttled/store/goredisstore"

	"github.com/miphilipp/devchat-server/internal/communication/server"
	"github.com/miphilipp/devchat-server/internal/communication/session"
	"github.com/miphilipp/devchat-server/internal/communication/websocket"
	"github.com/miphilipp/devchat-server/internal/database"
	"github.com/miphilipp/devchat-server/internal/mailing"
	"github.com/miphilipp/devchat-server/internal/conversations"
	"github.com/miphilipp/devchat-server/internal/messaging"
	"github.com/miphilipp/devchat-server/internal/user"
)

func main() {
	var wait time.Duration
	var verbose bool
	var configPath string

	flag.DurationVar(&wait, "graceful-timeout", time.Second * 15, "the duration for which the server gracefully wait for existing connections to finish - e.g. 15s or 1m")
	flag.BoolVar(&verbose, "verbose", false, "")
	flag.StringVar(&configPath, "configPath", "./config.yaml", "")
	flag.Parse()

	var logger log.Logger
	logger = log.NewLogfmtLogger(log.NewSyncWriter(os.Stderr))
	logger = log.With(logger, "ts", log.DefaultTimestampUTC)

	if verbose {
		logger = level.NewFilter(logger, level.AllowInfo())
	}

	level.Info(logger).Log("Message", "Starting up...")

	var cfg config
	err := readConfigFile(configPath, &cfg)
	if err != nil {
		level.Error(logger).Log(
			"System", "Configuration", 
			"Path", configPath,
			"err", err,
		)
		os.Exit(1)
	}

	level.Info(logger).Log("Addr", cfg.Server.Addr)

	var useSSL = (cfg.Server.CertFile != "" && cfg.Server.KeyFile != "")

	db, err := database.Connect(cfg.Database.Addr, cfg.Database.User, cfg.Database.Password, cfg.Database.Name)
	if err != nil {
		level.Error(logger).Log(
			"System", "Main-database", 
			"err", "DB Connection could not be established.",
			"address", cfg.Database.Addr,
		)
		os.Exit(1)
	}
	defer db.Close()

	messageRepo := database.NewMessageRepository(db)
	conversationRepo := database.NewConversationRepository(db)
	userRepo := database.NewUserRepository(db)

	var mailingService = mailing.NewService(
		cfg.Mailing.Server,
		cfg.Mailing.Port,
		cfg.Mailing.Password,
		cfg.Mailing.User,
		cfg.Mailing.MailAddr,
	)

	var userService user.Service
	userService = user.NewService(userRepo, mailingService)
	userService = user.NewLoggingService(logger, userService, verbose)

	var conversationService conversations.Service
	conversationService = conversations.NewService(conversationRepo)
	conversationService = conversations.NewLoggingService(logger, conversationService, verbose)

	var messagingService messaging.Service
	messagingService = messaging.NewService(messageRepo, conversationRepo)
	messagingService = messaging.NewLoggingService(logger, messagingService, verbose)


	sessionPersistance, err := session.NewInMemorySessionPersistance(cfg.InMemoryDB.Addr, cfg.InMemoryDB.Password)
	if err != nil {
		level.Error(logger).Log("System", "SessionPersistance", "err", err)
		os.Exit(1)
	}

	session := session.NewSessionManager(sessionPersistance)
	if err != nil {
		level.Error(logger).Log("System", "SessionManager", "err", err)
		os.Exit(1)
	}

	limiterStore, _ := limiter.New(sessionPersistance.RedisClient, "limiter_")

	socket := websocket.New(
		messagingService, 
		conversationService, 
		limiterStore,
		log.WithPrefix(logger, "Interface", "websocket"))
	if socket == nil {
		os.Exit(1)
	}

	app := server.New(
		cfg.Server.Addr, 
		cfg.Server.IndexFileName, 
		cfg.Server.AssetsFolder, 
		userService, 
		conversationService, 
		messagingService,
		socket,
		session,
		limiterStore,
		logger,
	)

	app.SetupRestHandlers()
	app.SetupFileServer()

	go func() {
		if !useSSL {
			err = app.Server.ListenAndServe()
		} else {
			err = app.Server.ListenAndServeTLS(cfg.Server.CertFile, cfg.Server.KeyFile)
		}
		logger.Log("Reason for quitting", err)
	}()


	var redirectServer *http.Server
	if useSSL {
		redirectServer = server.NewRedirectServer()
		go func() {
			err = redirectServer.ListenAndServe()
			logger.Log("Reason for quitting", err)
		}()
	}

	c := make(chan os.Signal, 1)
    signal.Notify(c, os.Interrupt)
    <-c

    ctx, cancel := context.WithTimeout(context.Background(), wait)
	defer cancel()
	app.Server.Shutdown(ctx)
	if redirectServer != nil { redirectServer.Shutdown(ctx) }
	err = sessionPersistance.Persist()
	if err != nil {
		level.Error(logger).Log("System", "SessionPersistance", "err", err)
	}
    // Optionally, you could run srv.Shutdown in a goroutine and block on
    // <-ctx.Done() if your application should wait for other services
    // to finalize based on context cancellation.
    level.Info(logger).Log("Messsage", "shutting down")
    os.Exit(0)
}
