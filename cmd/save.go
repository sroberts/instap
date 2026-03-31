package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/sroberts/instap/internal/api"
)

var saveCmd = &cobra.Command{
	Use:   "save [url]",
	Short: "Save a URL to Instapaper",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		bookmarkURL := args[0]
		if !api.IsValidURL(bookmarkURL) {
			fmt.Printf("Invalid URL: %s (Must start with http:// or https://)\n", bookmarkURL)
			return
		}

		consumerKey := viper.GetString("consumer_key")
		consumerSecret := viper.GetString("consumer_secret")
		accessToken := viper.GetString("access_token")
		accessSecret := viper.GetString("access_token_secret")

		if consumerKey == "" || consumerSecret == "" || accessToken == "" || accessSecret == "" {
			fmt.Println("Please log in first using 'instap auth login'")
			return
		}

		client := api.NewClient(consumerKey, consumerSecret, accessToken, accessSecret)
		bookmark, err := client.AddBookmark(bookmarkURL)
		if err != nil {
			fmt.Printf("Error saving bookmark: %v\n", err)
			return
		}

		fmt.Printf("Saved: %s\n", bookmark.Title)
	},
}

func init() {
	RootCmd.AddCommand(saveCmd)
}
