package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

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
It allows you to save bookmarks, list them, and manage them via an interactive TUI.
Piping data (URLs and optional tags) into instap will automatically save them.`,
	Run: func(cmd *cobra.Command, args []string) {
		// Check for piped input
		fi, _ := os.Stdin.Stat()
		if (fi.Mode() & os.ModeCharDevice) == 0 {
			saveFromStdin()
			return
		}

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

func saveFromStdin() {
	consumerKey := viper.GetString("consumer_key")
	consumerSecret := viper.GetString("consumer_secret")
	accessToken := viper.GetString("access_token")
	accessSecret := viper.GetString("access_token_secret")

	if consumerKey == "" || consumerSecret == "" || accessToken == "" || accessSecret == "" {
		fmt.Println("Please log in first using 'instap auth login'")
		return
	}

	client := api.NewClient(consumerKey, consumerSecret, accessToken, accessSecret)
	scanner := bufio.NewScanner(os.Stdin)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		parts := strings.Fields(line)
		bookmarkURL := parts[0]
		if !api.IsValidURL(bookmarkURL) {
			fmt.Printf("Skipping invalid URL: %s\n", bookmarkURL)
			continue
		}
		var tags []string
		if len(parts) > 1 {
			tags = strings.Split(parts[1], ",")
		}

		bookmark, err := client.AddBookmark(bookmarkURL)
		if err != nil {
			fmt.Printf("Error saving %s: %v\n", bookmarkURL, err)
			continue
		}

		if len(tags) > 0 {
			err = client.SetTags(bookmark.ID, tags)
			if err != nil {
				fmt.Printf("Error tagging %s: %v\n", bookmarkURL, err)
			}
		}

		fmt.Printf("Saved: %s\n", bookmark.Title)
	}
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
