package main

import (
	"context"
	"github.com/Shuttl-Tech/drone-autoscaler/cluster"
	"github.com/Shuttl-Tech/drone-autoscaler/config"
	"github.com/Shuttl-Tech/drone-autoscaler/engine"
	"github.com/drone/drone-go/drone"
	log "github.com/sirupsen/logrus"
	"golang.org/x/oauth2"
	"net/url"
	"os"
)

func main() {
	ctx := context.Background()

	conf, err := config.Load()
	if err != nil {
		panic(err)
	}

	setupLogging(conf)
	client := setupDroneClient(ctx, conf)
	fleet := setupAgentCluster(conf)

	log.Info("Starting autoscaler")
	engine.New(conf, client, fleet).Start(ctx)
}

func setupLogging(c config.Config) {
	log.SetOutput(os.Stdout)

	if c.LogFormat == "json" {
		log.SetFormatter(&log.JSONFormatter{})
	}

	if c.Debug {
		log.SetLevel(log.DebugLevel)
	} else {
		log.SetLevel(log.InfoLevel)
	}
}

func setupDroneClient(ctx context.Context, c config.Config) drone.Client {
	oauth2Config := new(oauth2.Config)
	authenticator := oauth2Config.Client(
		ctx,
		&oauth2.Token{
			AccessToken: c.Server.AuthToken,
		},
	)
	uri := new(url.URL)
	uri.Scheme = c.Server.Proto
	uri.Host = c.Server.Host
	return drone.NewClient(uri.String(), authenticator)
}

func setupAgentCluster(c config.Config) cluster.Cluster {
	return cluster.New(c.Agent.AutoscalingGroup)
}
