package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/sroberts/instap/internal/api"
	"github.com/sroberts/instap/internal/tui"
)

var cfgFile string

// RootCmd represents the base command when called without any subcommands
var RootCmd = &cobra.Command{
	Use:   "instap",
	Short: "A command-line tool for Instapaper",
	Long: `instap is a CLI tool for interacting with the Instapaper API.
It allows you to save bookmarks, list them, and manage them via an interactive TUI.`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			consumerKey := viper.GetString("consumer_key")
			consumerSecret := viper.GetString("consumer_secret")
			accessToken := viper.GetString("access_token")
			accessSecret := viper.GetString("access_token_secret")

			if consumerKey == "" || consumerSecret == "" || accessToken == "" || accessSecret == "" {
				fmt.Println("Please log in first using 'instap auth login'")
				return
			}

			client := api.NewClient(consumerKey, consumerSecret, accessToken, accessSecret)
			if err := tui.Run(client); err != nil {
				fmt.Printf("Error running TUI: %v\n", err)
				os.Exit(1)
			}
		}
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() {
	if err := RootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	RootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.instap.yaml)")
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		home, err := os.UserHomeDir()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		viper.AddConfigPath(home)
		viper.SetConfigName(".instap")
		viper.SetConfigType("yaml")
	}

	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err == nil {
		// fmt.Println("Using config file:", viper.ConfigFileUsed())
	}
}
