package cmd

import (
	"fmt"
	"os"
	"runtime"
	"time"

	"github.com/TFMV/blink/pkg/blink"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	// Used for flags
	cfgFile         string
	path            string
	allowedOrigin   string
	eventAddr       string
	eventPath       string
	refreshDuration time.Duration
	verbose         bool
	maxProcs        int

	// rootCmd represents the base command when called without any subcommands
	rootCmd = &cobra.Command{
		Use:   "blink",
		Short: "A high-performance file system watcher",
		Long: `Blink is a high-performance file system watcher that monitors 
directories for changes and provides events through a server-sent events (SSE) stream.

It can be used to trigger browser refreshes, run tests, or perform other actions
when files change.`,
		// Uncomment the following line if your bare application
		// has an action associated with it:
		RunE: func(cmd *cobra.Command, args []string) error {
			return runWatcher()
		},
	}
)

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.blink.yaml)")

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	rootCmd.Flags().StringVar(&path, "path", ".", "Directory path to watch for changes (must be a valid directory)")
	rootCmd.Flags().StringVar(&allowedOrigin, "allowed-origin", "*", "Value for Access-Control-Allow-Origin header")
	rootCmd.Flags().StringVar(&eventAddr, "event-addr", ":12345", "Address to serve events on ([host][:port])")
	rootCmd.Flags().StringVar(&eventPath, "event-path", "/events", "URL path for the event stream")
	rootCmd.Flags().DurationVar(&refreshDuration, "refresh", 100*time.Millisecond, "Refresh duration for events")
	rootCmd.Flags().BoolVar(&verbose, "verbose", false, "Enable verbose logging")
	rootCmd.Flags().IntVar(&maxProcs, "max-procs", runtime.NumCPU(), "Maximum number of CPUs to use")

	// Bind flags to viper
	viper.BindPFlag("path", rootCmd.Flags().Lookup("path"))
	viper.BindPFlag("allowed-origin", rootCmd.Flags().Lookup("allowed-origin"))
	viper.BindPFlag("event-addr", rootCmd.Flags().Lookup("event-addr"))
	viper.BindPFlag("event-path", rootCmd.Flags().Lookup("event-path"))
	viper.BindPFlag("refresh", rootCmd.Flags().Lookup("refresh"))
	viper.BindPFlag("verbose", rootCmd.Flags().Lookup("verbose"))
	viper.BindPFlag("max-procs", rootCmd.Flags().Lookup("max-procs"))

	// Set default values in viper
	viper.SetDefault("path", ".")
	viper.SetDefault("allowed-origin", "*")
	viper.SetDefault("event-addr", ":12345")
	viper.SetDefault("event-path", "/events")
	viper.SetDefault("refresh", 100*time.Millisecond)
	viper.SetDefault("verbose", false)
	viper.SetDefault("max-procs", runtime.NumCPU())
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := os.UserHomeDir()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		// Search config in home directory with name ".blink" (without extension).
		viper.AddConfigPath(home)
		viper.SetConfigType("yaml")
		viper.SetConfigName(".blink")
	}

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		fmt.Println("Using config file:", viper.ConfigFileUsed())
	}
}

// runWatcher starts the file watcher and event server
func runWatcher() error {
	// Set the maximum number of CPUs to use
	runtime.GOMAXPROCS(viper.GetInt("max-procs"))

	// Set verbose mode if requested
	blink.SetVerbose(viper.GetBool("verbose"))

	// Get the path to watch, ensuring it's a valid directory
	watchPath := viper.GetString("path")

	// Check if the path exists and is a directory
	fileInfo, err := os.Stat(watchPath)
	if err != nil {
		return fmt.Errorf("error accessing path %s: %w", watchPath, err)
	}

	if !fileInfo.IsDir() {
		return fmt.Errorf("path is not a directory: %s", watchPath)
	}

	// Print startup information
	fmt.Printf("Blink File System Watcher\n")
	fmt.Printf("-------------------------\n")
	fmt.Printf("Watching directory: %s\n", watchPath)
	fmt.Printf("Serving events at: http://%s%s\n", viper.GetString("event-addr"), viper.GetString("event-path"))
	fmt.Printf("Refresh duration: %v\n", viper.GetDuration("refresh"))
	fmt.Printf("Verbose mode: %v\n", viper.GetBool("verbose"))
	fmt.Printf("Using %d CPUs\n", viper.GetInt("max-procs"))
	fmt.Printf("Press Ctrl+C to exit\n\n")

	// Start the event server
	blink.EventServer(
		watchPath,
		viper.GetString("allowed-origin"),
		viper.GetString("event-addr"),
		viper.GetString("event-path"),
		viper.GetDuration("refresh"),
	)

	// Block forever (the event server runs in a goroutine)
	select {}
}
