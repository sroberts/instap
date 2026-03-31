package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/sroberts/instap/internal/api"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List recent bookmarks",
	Run: func(cmd *cobra.Command, args []string) {
		consumerKey := viper.GetString("consumer_key")
		consumerSecret := viper.GetString("consumer_secret")
		accessToken := viper.GetString("access_token")
		accessSecret := viper.GetString("access_token_secret")

		if consumerKey == "" || consumerSecret == "" || accessToken == "" || accessSecret == "" {
			fmt.Println("Please log in first using 'instap auth login'")
			return
		}

		client := api.NewClient(consumerKey, consumerSecret, accessToken, accessSecret)
		bookmarks, err := client.ListBookmarks("")
		if err != nil {
			fmt.Printf("Error listing bookmarks: %v\n", err)
			return
		}

		for _, b := range bookmarks {
			fmt.Printf("- %s (%s)\n", b.Title, b.URL)
		}
	},
}

func init() {
	RootCmd.AddCommand(listCmd)
}
