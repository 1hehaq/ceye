package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/charmbracelet/log"
	"github.com/jmoiron/jsonq"
)

const version = "1.1.0"

var (
	target      = flag.String("target", "", "target domain to monitor")
	configPath  = flag.String("config", "", "path to configuration file")
	notify      = flag.String("notify", "", "notification provider: discord, telegram, both")
	showVersion = flag.Bool("version", false, "show version")
	update      = flag.Bool("update", false, "update to latest version")
	showHelp    = flag.Bool("h", false, "show help")
	showHelp2   = flag.Bool("help", false, "show help")
	logger      = log.NewWithOptions(os.Stderr, log.Options{
		ReportTimestamp: true,
		TimeFormat:      "15:04:05",
	})
	targets        []string
	webhookURL     string
	telegramToken  string
	telegramChatID string
	dockerManager  *DockerManager
	notifyDiscord  bool
	notifyTelegram bool
)

func main() {
	flag.Parse()

	// Check for unrecognized flags
	if len(os.Args) > 1 {
		validFlags := map[string]bool{
			"-target": true, "-config": true, "-notify": true,
			"-version": true, "-update": true, "-h": true, "-help": true,
		}

		for _, arg := range os.Args[1:] {
			if strings.HasPrefix(arg, "-") && !validFlags[arg] {
				displayHelp()
				return
			}
		}
	}

	if *showVersion {
		displayVersion()
		return
	}

	if *update {
		performUpdate()
		return
	}

	if *showHelp || *showHelp2 {
		displayHelp()
		return
	}

	printBanner()

	if *configPath != "" {
		setConfigPath(*configPath)
	}

	cfg, err := loadConfig()
	if err != nil {
		logger.Fatal("failed to load config", "error", err)
	}

	if cfg != nil {
		if cfg.Webhook == `""` {
			cfg.Webhook = ""
		}

		webhookURL = strings.TrimSpace(cfg.Webhook)
		if webhookURL == "" {
			logger.Warn("no discord webhook configured in configuration file; discord notifications disabled")
		}

		telegramToken = strings.TrimSpace(cfg.TelegramBotToken)
		telegramChatID = strings.TrimSpace(cfg.TelegramChatID)
	} else {
		webhookURL = ""
		telegramToken = ""
		telegramChatID = ""
		logger.Warn("no configuration file found. notifications will be disabled unless providers are configured")
	}

	stdinAvailable := false
	if fi, err := os.Stdin.Stat(); err == nil && (fi.Mode()&os.ModeCharDevice) == 0 {
		stdinAvailable = true
	}

	switch {
	case *target != "":
		resolved, err := resolveTargetFlag(*target)
		if err != nil {
			logger.Fatal("failed to resolve target", "error", err)
		}
		if len(resolved) == 0 {
			logger.Fatal("no targets resolved from -target flag")
		}
		targets = resolved
		logger.Info("using targets from cli flag", "count", len(targets))
	case stdinAvailable:
		resolved, err := loadTargetsFromStdin()
		if err != nil {
			logger.Fatal("failed to read targets from stdin", "error", err)
		}
		if len(resolved) == 0 {
			logger.Fatal("no targets provided on stdin")
		}
		targets = resolved
		logger.Info("using targets from stdin", "count", len(targets))
	case cfg != nil:
		if len(cfg.Targets) == 0 {
			logger.Fatal("no targets configured. please add target domains to ~/.config/ceye/provider.yaml or use -target flag or stdin")
		}
		targets = cfg.Targets
		logger.Info("loaded configuration", "targets", len(targets))
	default:
		if err := createConfigTemplate(); err != nil {
			logger.Fatal("failed to create config template", "error", err)
		}
		configPath, _ := getConfigPath()
		logger.Info("created config template", "path", configPath)
		logger.Fatal("please edit the configuration file or provide targets via -target or stdin and run again")
	}

	discordConfigured := webhookURL != ""
	telegramConfigured := telegramToken != "" && telegramChatID != ""

	notifyValue := strings.ToLower(strings.TrimSpace(*notify))
	switch notifyValue {
	case "":
		if discordConfigured && telegramConfigured {
			notifyDiscord = true
			notifyTelegram = true
		} else if discordConfigured {
			notifyDiscord = true
		} else if telegramConfigured {
			notifyTelegram = true
		} else {
			logger.Fatal("no notification provider configured. please configure a discord webhook or telegram bot token/chat id")
		}
	case "discord":
		if !discordConfigured {
			logger.Fatal("notify=discord selected but discord webhook is not configured. please configure it in your configuration file (use -config for a custom path)")
		}
		notifyDiscord = true
	case "telegram":
		if !telegramConfigured {
			logger.Fatal("notify=telegram selected but telegram bot token/chat id are not configured. please configure them in your configuration file (use -config for a custom path)")
		}
		notifyTelegram = true
	case "both":
		if !discordConfigured && !telegramConfigured {
			logger.Fatal("notify=both selected but neither discord nor telegram is configured")
		}
		if !discordConfigured {
			logger.Warn("notify=both selected but discord webhook is not configured; falling back to telegram only")
		}
		if !telegramConfigured {
			logger.Warn("notify=both selected but telegram bot token/chat id are not configured; falling back to discord only")
		}
		notifyDiscord = discordConfigured
		notifyTelegram = telegramConfigured
	default:
		logger.Fatal("invalid value for -notify. valid options are: discord, telegram, both")
	}

	ctx, cancel := context.WithCancel(context.Background())
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		logger.Info("shutting down...")
		cancel()
	}()

	dockerManager = NewDockerManager()

	logger.Info("initializing certstream server")
	if err := dockerManager.EnsureRunning(ctx); err != nil {
		logger.Fatal("failed to start certstream server", "error", err)
	}

	logger.Info("starting ceye")
	for i, t := range targets {
		fmt.Printf("         %d. %s\n", (i + 1), t)
	}

	wsURL := dockerManager.GetWebSocketURL()
	logger.Info("connecting to certstream", "url", wsURL)

	stream, errStream := CertStreamEventStream(wsURL)

	for {
		select {
		case <-ctx.Done():
			logger.Info("goodbye")
			return
		case jq := <-stream:
			processMessage(jq)
		case err := <-errStream:
			if err != nil {
				logger.Warn("certstream error", "error", err.Error())
			}
		}
	}
}

func resolveTargetFlag(value string) ([]string, error) {
	if value == "-" {
		return loadTargetsFromStdin()
	}

	if info, err := os.Stat(value); err == nil && !info.IsDir() {
		file, err := os.Open(value)
		if err != nil {
			return nil, err
		}
		defer file.Close()

		scanner := bufio.NewScanner(file)
		var targets []string
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			targets = append(targets, line)
		}
		if err := scanner.Err(); err != nil {
			return nil, err
		}

		return targets, nil
	}

	return []string{value}, nil
}

func loadTargetsFromStdin() ([]string, error) {
	scanner := bufio.NewScanner(os.Stdin)
	var targets []string
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		targets = append(targets, line)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return targets, nil
}

func processMessage(jq jsonq.JsonQuery) {
	messageType, err := jq.String("message_type")
	if err != nil || messageType != "certificate_update" {
		return
	}

	domains, err := jq.ArrayOfStrings("data", "leaf_cert", "all_domains")
	if err != nil {
		return
	}

	for _, domain := range domains {
		for _, target := range targets {
			if strings.Contains(strings.ToLower(domain), strings.ToLower(target)) {
				logger.Info("new subdomain", "domain", domain, "target", target)
				if notifyDiscord || notifyTelegram {
					go sendToDiscord(domain, target)
				}
				break
			}
		}
	}
}
