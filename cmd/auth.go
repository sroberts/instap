package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/sroberts/instap/internal/api"
)

var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Manage Instapaper authentication",
}

var setCredentialsCmd = &cobra.Command{
	Use:   "set-credentials",
	Short: "Set OAuth consumer key and secret",
	Run: func(cmd *cobra.Command, args []string) {
		key, _ := cmd.Flags().GetString("key")
		secret, _ := cmd.Flags().GetString("secret")

		if key == "" || secret == "" {
			fmt.Println("Both consumer key and secret are required.")
			return
		}

		viper.Set("consumer_key", key)
		viper.Set("consumer_secret", secret)

		if err := viper.WriteConfig(); err != nil {
			// If config doesn't exist, create it.
			home, _ := os.UserHomeDir()
			configPath := home + "/.instap.yaml"
			if err := viper.WriteConfigAs(configPath); err != nil {
				fmt.Printf("Error saving config: %v\n", err)
				return
			}
			fmt.Printf("Config created and saved to %s\n", configPath)
		} else {
			fmt.Println("Credentials updated successfully.")
		}
	},
}

var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Log in to Instapaper with your username and password",
	Run: func(cmd *cobra.Command, args []string) {
		consumerKey := viper.GetString("consumer_key")
		consumerSecret := viper.GetString("consumer_secret")

		if consumerKey == "" || consumerSecret == "" {
			fmt.Println("Please set consumer credentials first using 'instap auth set-credentials'")
			return
		}

		var username, password string
		fmt.Print("Username (email): ")
		fmt.Scanln(&username)
		fmt.Print("Password: ")
		// Note: In a real app, use a package like 'gopass' for secure password entry
		fmt.Scanln(&password)

		token, secret, err := api.GetAccessToken(consumerKey, consumerSecret, username, password)
		if err != nil {
			fmt.Printf("Login failed: %v\n", err)
			return
		}

		viper.Set("access_token", token)
		viper.Set("access_token_secret", secret)

		if err := viper.WriteConfig(); err != nil {
			fmt.Printf("Error saving tokens: %v\n", err)
			return
		}

		fmt.Println("Logged in successfully.")
	},
}

func init() {
	RootCmd.AddCommand(authCmd)
	authCmd.AddCommand(setCredentialsCmd)
	authCmd.AddCommand(loginCmd)

	setCredentialsCmd.Flags().StringP("key", "k", "", "OAuth consumer key")
	setCredentialsCmd.Flags().StringP("secret", "s", "", "OAuth consumer secret")
}
