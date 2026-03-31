package cmd

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/sroberts/instap/internal/api"
)

var tagCmd = &cobra.Command{
	Use:   "tag",
	Short: "Manage tags for a bookmark",
}

var tagAddCmd = &cobra.Command{
	Use:   "add [bookmark_id] [tag1,tag2,...]",
	Short: "Add tags to a bookmark",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		id, err := strconv.Atoi(args[0])
		if err != nil {
			fmt.Printf("Invalid bookmark ID: %s\n", args[0])
			return
		}
		tags := strings.Split(args[1], ",")

		client := getAPIClient()
		if client == nil {
			return
		}

		err = client.AddTags(id, tags)
		if err != nil {
			fmt.Printf("Error adding tags: %v\n", err)
			return
		}
		fmt.Println("Tags added successfully.")
	},
}

var tagRemoveCmd = &cobra.Command{
	Use:   "remove [bookmark_id] [tag1,tag2,...]",
	Short: "Remove tags from a bookmark",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		id, err := strconv.Atoi(args[0])
		if err != nil {
			fmt.Printf("Invalid bookmark ID: %s\n", args[0])
			return
		}
		tags := strings.Split(args[1], ",")

		client := getAPIClient()
		if client == nil {
			return
		}

		err = client.RemoveTags(id, tags)
		if err != nil {
			fmt.Printf("Error removing tags: %v\n", err)
			return
		}
		fmt.Println("Tags removed successfully.")
	},
}

var tagSetCmd = &cobra.Command{
	Use:   "set [bookmark_id] [tag1,tag2,...]",
	Short: "Set (overwrite) tags for a bookmark",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		id, err := strconv.Atoi(args[0])
		if err != nil {
			fmt.Printf("Invalid bookmark ID: %s\n", args[0])
			return
		}
		tags := strings.Split(args[1], ",")

		client := getAPIClient()
		if client == nil {
			return
		}

		err = client.SetTags(id, tags)
		if err != nil {
			fmt.Printf("Error setting tags: %v\n", err)
			return
		}
		fmt.Println("Tags set successfully.")
	},
}

func getAPIClient() *api.Client {
	consumerKey := viper.GetString("consumer_key")
	consumerSecret := viper.GetString("consumer_secret")
	accessToken := viper.GetString("access_token")
	accessSecret := viper.GetString("access_token_secret")

	if consumerKey == "" || consumerSecret == "" || accessToken == "" || accessSecret == "" {
		fmt.Println("Please log in first using 'instap auth login'")
		return nil
	}

	return api.NewClient(consumerKey, consumerSecret, accessToken, accessSecret)
}

func init() {
	RootCmd.AddCommand(tagCmd)
	tagCmd.AddCommand(tagAddCmd)
	tagCmd.AddCommand(tagRemoveCmd)
	tagCmd.AddCommand(tagSetCmd)
}
