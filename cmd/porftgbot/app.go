package main

import (
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/tdakkota/porftgbot/runner"

	"github.com/gotd/td/telegram"
	"github.com/gotd/td/tg"
	"github.com/urfave/cli/v2"
	"github.com/urfave/cli/v2/altsrc"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"golang.org/x/xerrors"

	"github.com/tdakkota/porftgbot/bot"
)

type App struct {
	logger *zap.Logger
	bot    *bot.Bot
}

func NewApp() *App {
	logger, _ := zap.NewDevelopment(zap.IncreaseLevel(zapcore.DebugLevel))
	return &App{
		logger: logger,
	}
}

func (app *App) createTelegram(c *cli.Context, dispatcher tg.UpdateDispatcher) (*telegram.Client, error) {
	logger := app.logger

	sessionDir := ""
	if c.IsSet("tg.session_dir") {
		sessionDir = c.String("tg.session_dir")
	} else {
		home, err := os.UserHomeDir()
		if err != nil {
			sessionDir = "./.td"
		} else {
			sessionDir = filepath.Join(home, ".td")
		}
	}
	if err := os.MkdirAll(sessionDir, 0600); err != nil {
		return nil, xerrors.Errorf("failed to create session dir: %w", err)
	}

	client := telegram.NewClient(c.Int("tg.app_id"), c.String("tg.app_hash"), telegram.Options{
		Logger: logger,
		SessionStorage: &telegram.FileSessionStorage{
			Path: filepath.Join(sessionDir, "session.json"),
		},
		UpdateHandler: dispatcher.Handle,
	})

	err := client.Connect(c.Context)
	if err != nil {
		return nil, xerrors.Errorf("failed to connect: %w", err)
	}

	auth, err := client.AuthStatus(c.Context)
	if err != nil {
		return nil, xerrors.Errorf("failed to get auth status: %w", err)
	}

	logger.With(zap.Bool("authorized", auth.Authorized)).Info("Auth status")
	if !auth.Authorized {
		if err := client.AuthBot(c.Context, c.String("tg.bot_token")); err != nil {
			return nil, xerrors.Errorf("failed to perform bot login: %w", err)
		}
		logger.Info("Bot login ok")
	}

	u, err := client.Self(c.Context)
	if err != nil {
		return nil, xerrors.Errorf("failed to ping: %w", err)
	}

	logger.With(zap.String("user", u.Username), zap.Bool("is_bot", u.Bot)).
		Info("Logged in")

	return client, nil
}

func (app *App) run(c *cli.Context) error {
	dispatcher := tg.NewUpdateDispatcher()
	client, err := app.createTelegram(c, dispatcher)
	if err != nil {
		return err
	}

	app.bot = bot.NewBot(
		runner.NewHTTPRunner(),
		tg.NewClient(client),
		app.logger.Named("bot"),
	)
	dispatcher.OnBotInlineQuery(app.bot.Handler())

	// Reading updates until SIGTERM.
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)

	<-sig
	app.logger.Info("Shutting down")
	if err := client.Close(); err != nil {
		return err
	}
	app.logger.Info("Graceful shutdown completed")
	return nil
}

func (app *App) getEnvNames(names ...string) []string {
	return names
}

func (app *App) flags() []cli.Flag {
	flags := []cli.Flag{
		&cli.StringFlag{
			Name:    "config.file",
			Value:   "porftgbot.yml",
			Usage:   "path to config file",
			EnvVars: app.getEnvNames("CONFIG_FILE", "CONFIG"),
		},

		// tg
		altsrc.NewIntFlag(&cli.IntFlag{
			Name:     "tg.app_id",
			Required: true,
			Usage:    "Telegram app ID",
			Aliases:  []string{"app_id"},
			EnvVars:  app.getEnvNames("APP_ID"),
		}),
		altsrc.NewStringFlag(&cli.StringFlag{
			Name:     "tg.app_hash",
			Required: true,
			Usage:    "Telegram app hash",
			Aliases:  []string{"app_hash"},
			EnvVars:  app.getEnvNames("APP_HASH"),
		}),
		altsrc.NewStringFlag(&cli.StringFlag{
			Name:     "tg.bot_token",
			Required: true,
			Usage:    "Telegram bot token",
			Aliases:  []string{"token"},
			EnvVars:  app.getEnvNames("BOT_TOKEN"),
		}),
		altsrc.NewPathFlag(&cli.PathFlag{
			Name:    "tg.session_dir",
			Usage:   "Telegram session dir",
			Aliases: []string{"session_dir"},
			EnvVars: app.getEnvNames("SESSION_DIR"),
		}),
	}

	return flags
}

func (app *App) commands() []*cli.Command {
	commands := []*cli.Command{
		{
			Name:        "run",
			Description: "runs bot",
			Flags:       app.flags(),
			Action:      app.run,
		},
	}

	app.addFileConfig("config.file", commands[0])
	return commands
}

func (app *App) addFileConfig(flagName string, command *cli.Command) {
	prev := command.Before

	command.Before = func(context *cli.Context) error {
		if prev != nil {
			err := prev(context)
			if err != nil {
				return err
			}
		}

		path := context.String(flagName)
		fileContext, err := altsrc.NewYamlSourceFromFile(path)
		if err != nil {
			app.logger.Info("failed to load config from", zap.String("path", path))
			return nil
		}

		return altsrc.ApplyInputSourceValues(context, fileContext, command.Flags)
	}
}

func (app *App) cli() *cli.App {
	cliApp := &cli.App{
		Name:     "porftgbot",
		Usage:    "Telegram Bot for https://porfirevich.ru/",
		Commands: app.commands(),
	}

	return cliApp
}

func (app *App) Run(args []string) error {
	return app.cli().Run(args)
}

func main() {
	if err := NewApp().Run(os.Args); err != nil {
		_, _ = os.Stdout.WriteString(err.Error() + "\n")
		os.Exit(1)
	}
}
