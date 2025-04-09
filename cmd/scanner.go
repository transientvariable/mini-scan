package cmd

import (
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/spf13/cobra"
	"github.com/transientvariable/log-go"
)

var (
	logLevel         string
	projectID        string
	subscriptionName string
	topicID          string
)

func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "scanner [flags]>",
		Long:  "CLI for running scanner.",
		Short: "scanner CLI",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := log.SetDefault(log.New(log.WithLevel(logLevel))); err != nil {
				return fmt.Errorf("cmd_scanner: %w", err)
			}

			d, err := newDispatcher(projectID, topicID, subscriptionName)
			if err != nil {
				return fmt.Errorf("cmd_scanner: %w", err)
			}

			if err := d.Run(); err != nil {
				return fmt.Errorf("cmd_scanner: %w", err)
			}

			var wg sync.WaitGroup
			wg.Add(1)

			c := make(chan os.Signal)
			signal.Notify(c, os.Interrupt, syscall.SIGTERM)
			go func() {
				<-c
				if err := d.Close(); err != nil {
					log.Error("[cmd:scanner]", log.Err(err))
				}
				wg.Done()
				os.Exit(1)
			}()
			wg.Wait()

			log.Info("[cmd:scanner] shut down...")
			return nil
		},
	}

	if err := log.SetDefault(log.New(log.WithLevel("info"))); err != nil {
		panic(err)
	}

	f := cmd.Flags()
	f.StringVarP(&logLevel, "log-level", "l", "info", "Sets the logging level.")
	f.StringVarP(&projectID, "project-id", "p", "test-project", "Sets the GCP project ID.")
	f.StringVarP(&topicID, "topic-id", "t", "scan-topic", "Sets the GCP topic ID.")
	f.StringVarP(&subscriptionName, "subscription-name", "s", "scan-sub", "Sets the GCP subscription name.")
	return cmd
}
