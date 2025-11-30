package main

import (
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

const version = "0.0.1"

var (
	target      = flag.String("target", "", "target domain to monitor")
	webhook     = flag.String("webhook", "", "discord webhook URL")
	showVersion = flag.Bool("version", false, "show version")
	update      = flag.Bool("update", false, "update to latest version")
	showHelp    = flag.Bool("h", false, "show help")
	showHelp2   = flag.Bool("help", false, "show help")
	logger      = log.NewWithOptions(os.Stderr, log.Options{
		ReportTimestamp: true,
		TimeFormat:      "15:04:05",
	})
	domainCache   *DomainCache
	targets       []string
	webhookURL    string
	dockerManager *DockerManager
)

func main() {
	flag.Parse()

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

	cfg, err := loadConfig()
	if err != nil {
		logger.Fatal("failed to load config", "error", err)
	}

	if cfg != nil {
		if *webhook != "" {
			if err := updateWebhook(*webhook); err != nil {
				logger.Fatal("failed to update webhook", "error", err)
			}
			logger.Info("updated webhook in config file")
			cfg.Webhook = *webhook
		}

		if cfg.Webhook == "" || cfg.Webhook == `""` {
			logger.Fatal("webhook not configured. please add your discord webhook url to ~/.config/ceye/provider.yaml or use -webhook flag")
		}

		webhookURL = cfg.Webhook

		if *target != "" {
			targets = []string{*target}
			logger.Info("using target from cli flag", "target", *target)
		} else {
			if len(cfg.Targets) == 0 {
				logger.Fatal("no targets configured. please add target domains to ~/.config/ceye/provider.yaml or use -target flag")
			}
			targets = cfg.Targets
			logger.Info("loaded configuration", "targets", len(targets))
		}
	} else {
		if *target == "" {
			if err := createConfigTemplate(); err != nil {
				logger.Fatal("failed to create config template", "error", err)
			}
			configPath, _ := getConfigPath()
			logger.Info("created config template", "path", configPath)
			logger.Fatal("please edit the configuration file and run again")
		}
		targets = []string{*target}
		webhookURL = *webhook
		if webhookURL == "" {
			logger.Warn("no discord webhook provided. notifications disabled")
		}
	}

	domainCache, err = NewDomainCache()
	if err != nil {
		logger.Fatal("failed to initialize domain cache", "error", err)
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
				if domainCache.IsNew(domain) {
					logger.Info("new subdomain", "domain", domain, "target", target)
					if err := domainCache.Add(domain); err != nil {
						logger.Error("failed to cache domain", "error", err)
					}
					if webhookURL != "" {
						go sendToDiscord(domain, target)
					}
				}
				break
			}
		}
	}
}
