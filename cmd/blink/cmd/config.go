package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// configCmd represents the config command
var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage Blink configuration",
	Long:  `View and modify Blink configuration settings.`,
}

// configGetCmd represents the config get command
var configGetCmd = &cobra.Command{
	Use:   "get [key]",
	Short: "Get a configuration value",
	Long:  `Get the value of a configuration key.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		key := args[0]
		if viper.IsSet(key) {
			fmt.Printf("%v\n", viper.Get(key))
		} else {
			fmt.Printf("Key '%s' not found in configuration\n", key)
			os.Exit(1)
		}
	},
}

// configSetCmd represents the config set command
var configSetCmd = &cobra.Command{
	Use:   "set [key] [value]",
	Short: "Set a configuration value",
	Long:  `Set the value of a configuration key.`,
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		key := args[0]
		value := args[1]

		// Set the value in viper
		viper.Set(key, value)

		// Save the configuration
		if err := viper.WriteConfig(); err != nil {
			if os.IsNotExist(err) {
				// Config file does not exist, create it
				if err := viper.SafeWriteConfig(); err != nil {
					fmt.Printf("Error creating config file: %s\n", err)
					os.Exit(1)
				}
			} else {
				fmt.Printf("Error writing config file: %s\n", err)
				os.Exit(1)
			}
		}

		fmt.Printf("Set '%s' to '%s'\n", key, value)
	},
}

// configListCmd represents the config list command
var configListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all configuration values",
	Long:  `List all configuration values.`,
	Run: func(cmd *cobra.Command, args []string) {
		settings := viper.AllSettings()
		keys := make([]string, 0, len(settings))

		// Get all keys
		for k := range settings {
			keys = append(keys, k)
		}

		// Sort keys
		// sort.Strings(keys)

		// Print keys and values
		for _, k := range keys {
			fmt.Printf("%s: %v\n", k, viper.Get(k))
		}
	},
}

func init() {
	rootCmd.AddCommand(configCmd)
	configCmd.AddCommand(configGetCmd)
	configCmd.AddCommand(configSetCmd)
	configCmd.AddCommand(configListCmd)
}
