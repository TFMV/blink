package cmd

import (
	"fmt"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/TFMV/blink/pkg/blink"
	"github.com/TFMV/blink/pkg/logger"
	"github.com/rs/zerolog"
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
	showEvents      bool
	// Filtering flags
	includePatterns string
	excludePatterns string
	includeEvents   string
	ignoreEvents    string
	filterDev       bool
	// Webhook flags
	webhookURL              string
	webhookMethod           string
	webhookHeaders          string
	webhookTimeout          time.Duration
	webhookDebounceDuration time.Duration
	webhookMaxRetries       int
	// Streaming flags
	streamMethod string
	// Logging flags
	logLevel  string
	logPretty bool
	logColors bool

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
	rootCmd.Flags().BoolVar(&showEvents, "show-events", true, "Show file events in the console")
	rootCmd.Flags().IntVar(&maxProcs, "max-procs", runtime.NumCPU(), "Maximum number of CPUs to use")
	rootCmd.Flags().StringVar(&includePatterns, "include", "", "Include patterns for files (e.g., \"*.js,*.css,*.html\")")
	rootCmd.Flags().StringVar(&excludePatterns, "exclude", "", "Exclude patterns for files (e.g., \"node_modules,*.tmp\")")
	rootCmd.Flags().StringVar(&includeEvents, "events", "", "Include event types (e.g., \"write,create\")")
	rootCmd.Flags().StringVar(&ignoreEvents, "ignore", "", "Ignore event types (e.g., \"chmod\")")
	rootCmd.Flags().BoolVar(&filterDev, "filter-dev", false, "Filter out development-related noise")
	rootCmd.Flags().StringVar(&webhookURL, "webhook-url", "", "URL for the webhook")
	rootCmd.Flags().StringVar(&webhookMethod, "webhook-method", "POST", "HTTP method for the webhook")
	rootCmd.Flags().StringVar(&webhookHeaders, "webhook-headers", "", "Headers for the webhook")
	rootCmd.Flags().DurationVar(&webhookTimeout, "webhook-timeout", 5*time.Second, "Timeout for the webhook")
	rootCmd.Flags().DurationVar(&webhookDebounceDuration, "webhook-debounce-duration", 0*time.Second, "Debounce duration for the webhook")
	rootCmd.Flags().IntVar(&webhookMaxRetries, "webhook-max-retries", 3, "Maximum number of retries for the webhook")
	rootCmd.Flags().StringVar(&streamMethod, "stream-method", "sse", "Method for streaming events (sse, websocket, both)")
	// Add logging flags
	rootCmd.Flags().StringVar(&logLevel, "log-level", "info", "Log level (debug, info, warn, error, fatal)")
	rootCmd.Flags().BoolVar(&logPretty, "log-pretty", true, "Enable pretty logging")
	rootCmd.Flags().BoolVar(&logColors, "log-colors", true, "Enable colors in pretty logging")

	// Bind flags to viper
	viper.BindPFlag("path", rootCmd.Flags().Lookup("path"))
	viper.BindPFlag("allowed-origin", rootCmd.Flags().Lookup("allowed-origin"))
	viper.BindPFlag("event-addr", rootCmd.Flags().Lookup("event-addr"))
	viper.BindPFlag("event-path", rootCmd.Flags().Lookup("event-path"))
	viper.BindPFlag("refresh", rootCmd.Flags().Lookup("refresh"))
	viper.BindPFlag("verbose", rootCmd.Flags().Lookup("verbose"))
	viper.BindPFlag("show-events", rootCmd.Flags().Lookup("show-events"))
	viper.BindPFlag("max-procs", rootCmd.Flags().Lookup("max-procs"))
	viper.BindPFlag("include", rootCmd.Flags().Lookup("include"))
	viper.BindPFlag("exclude", rootCmd.Flags().Lookup("exclude"))
	viper.BindPFlag("events", rootCmd.Flags().Lookup("events"))
	viper.BindPFlag("ignore", rootCmd.Flags().Lookup("ignore"))
	viper.BindPFlag("filter-dev", rootCmd.Flags().Lookup("filter-dev"))
	viper.BindPFlag("webhook-url", rootCmd.Flags().Lookup("webhook-url"))
	viper.BindPFlag("webhook-method", rootCmd.Flags().Lookup("webhook-method"))
	viper.BindPFlag("webhook-headers", rootCmd.Flags().Lookup("webhook-headers"))
	viper.BindPFlag("webhook-timeout", rootCmd.Flags().Lookup("webhook-timeout"))
	viper.BindPFlag("webhook-debounce-duration", rootCmd.Flags().Lookup("webhook-debounce-duration"))
	viper.BindPFlag("webhook-max-retries", rootCmd.Flags().Lookup("webhook-max-retries"))
	viper.BindPFlag("stream-method", rootCmd.Flags().Lookup("stream-method"))
	viper.BindPFlag("log-level", rootCmd.Flags().Lookup("log-level"))
	viper.BindPFlag("log-pretty", rootCmd.Flags().Lookup("log-pretty"))
	viper.BindPFlag("log-colors", rootCmd.Flags().Lookup("log-colors"))

	// Set default values in viper
	viper.SetDefault("path", ".")
	viper.SetDefault("allowed-origin", "*")
	viper.SetDefault("event-addr", ":12345")
	viper.SetDefault("event-path", "/events")
	viper.SetDefault("refresh", 100*time.Millisecond)
	viper.SetDefault("verbose", false)
	viper.SetDefault("show-events", true)
	viper.SetDefault("max-procs", runtime.NumCPU())
	viper.SetDefault("include", "")
	viper.SetDefault("exclude", "")
	viper.SetDefault("events", "")
	viper.SetDefault("ignore", "")
	viper.SetDefault("filter-dev", false)
	viper.SetDefault("webhook-url", "")
	viper.SetDefault("webhook-method", "POST")
	viper.SetDefault("webhook-headers", "")
	viper.SetDefault("webhook-timeout", 5*time.Second)
	viper.SetDefault("webhook-debounce-duration", 0*time.Second)
	viper.SetDefault("webhook-max-retries", 3)
	viper.SetDefault("stream-method", "sse")
	viper.SetDefault("log-level", "info")
	viper.SetDefault("log-pretty", true)
	viper.SetDefault("log-colors", true)
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

	// Use specific environment variable prefixes to avoid conflicts
	viper.SetEnvPrefix("BLINK")

	// Replace dots and dashes in flags with underscores for environment variables
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_", ".", "_"))

	// Read in environment variables that match
	viper.AutomaticEnv()

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		fmt.Println("Using config file:", viper.ConfigFileUsed())
	}

	// Initialize logger
	initLogger()

	// Set GOMAXPROCS
	runtime.GOMAXPROCS(viper.GetInt("max-procs"))
}

// initLogger initializes the logger with the configured options
func initLogger() {
	// Parse log level
	level := zerolog.InfoLevel
	switch strings.ToLower(viper.GetString("log-level")) {
	case "debug":
		level = zerolog.DebugLevel
	case "info":
		level = zerolog.InfoLevel
	case "warn":
		level = zerolog.WarnLevel
	case "error":
		level = zerolog.ErrorLevel
	case "fatal":
		level = zerolog.FatalLevel
	}

	// Initialize logger
	logger.Init(
		logger.WithLevel(level),
		logger.WithPretty(viper.GetBool("log-pretty")),
		logger.WithColors(viper.GetBool("log-colors")),
	)

	// Set verbose mode
	blink.SetVerbose(viper.GetBool("verbose"))
}

// runWatcher starts the file watcher and event server
func runWatcher() error {
	// Set the maximum number of CPUs to use
	runtime.GOMAXPROCS(viper.GetInt("max-procs"))

	// Set verbose mode if requested
	blink.SetVerbose(viper.GetBool("verbose"))

	// Get the path to watch, ensuring it's a valid directory
	watchPath := viper.GetString("path")

	// If path is empty or equals ".", use the current working directory
	if watchPath == "" || watchPath == "." {
		var err error
		watchPath, err = os.Getwd()
		if err != nil {
			return fmt.Errorf("error getting current working directory: %w", err)
		}
		fmt.Printf("Using current working directory: %s\n", watchPath)
	} else {
		fmt.Printf("Using specified path: %s\n", watchPath)
	}

	// Check if the path exists and is a directory
	fileInfo, err := os.Stat(watchPath)
	if err != nil {
		return fmt.Errorf("error accessing path %s: %w", watchPath, err)
	}

	if !fileInfo.IsDir() {
		return fmt.Errorf("path is not a directory: %s", watchPath)
	}

	// Prepare options
	var options []blink.Option

	// Create filter if any filter options are specified
	if viper.GetString("include") != "" || viper.GetString("exclude") != "" ||
		viper.GetString("events") != "" || viper.GetString("ignore") != "" ||
		viper.GetBool("filter-dev") {

		filter := blink.NewEventFilter()

		// Add include patterns if specified
		if includePatterns := viper.GetString("include"); includePatterns != "" {
			filter.SetIncludePatterns(includePatterns)
		}

		// Add exclude patterns if specified
		if excludePatterns := viper.GetString("exclude"); excludePatterns != "" {
			filter.SetExcludePatterns(excludePatterns)
		}

		// Add include events if specified
		if includeEvents := viper.GetString("events"); includeEvents != "" {
			filter.SetIncludeEvents(includeEvents)
		}

		// Add ignore events if specified
		if ignoreEvents := viper.GetString("ignore"); ignoreEvents != "" {
			filter.SetIgnoreEvents(ignoreEvents)
		}

		// Apply development filtering if enabled
		if viper.GetBool("filter-dev") {
			blink.ApplyDevFilter(filter, watchPath)
		}

		// Add the filter to options
		options = append(options, blink.WithFilter(filter))
	}

	// Add webhook options if specified
	if webhookURL := viper.GetString("webhook-url"); webhookURL != "" {
		options = append(options, blink.WithWebhook(
			webhookURL,
			viper.GetString("webhook-method"),
			parseHeaders(viper.GetString("webhook-headers")),
			viper.GetDuration("webhook-timeout"),
			viper.GetDuration("webhook-debounce-duration"),
			viper.GetInt("webhook-max-retries"),
		))
	}

	// Add stream method option
	streamMethodStr := viper.GetString("stream-method")
	var streamMethod blink.StreamMethod
	switch strings.ToLower(streamMethodStr) {
	case "sse":
		streamMethod = blink.StreamMethodSSE
	case "websocket":
		streamMethod = blink.StreamMethodWebSocket
	case "both":
		streamMethod = blink.StreamMethodBoth
	default:
		streamMethod = blink.StreamMethodSSE
	}
	options = append(options, blink.WithStreamMethod(streamMethod))

	// Add show events option
	options = append(options, blink.WithShowEvents(viper.GetBool("show-events")))

	// Print information about the watcher
	fmt.Printf("Watching %s\n", watchPath)
	fmt.Printf("Event server address: %s\n", viper.GetString("event-addr"))
	fmt.Printf("Event path: %s\n", viper.GetString("event-path"))
	fmt.Printf("Stream method: %s\n", streamMethodStr)
	fmt.Printf("Refresh duration: %v\n", viper.GetDuration("refresh"))
	fmt.Printf("Allowed origin: %s\n", viper.GetString("allowed-origin"))
	fmt.Printf("Show events: %v\n", viper.GetBool("show-events"))

	// Print filter information if specified
	if viper.GetString("include") != "" {
		fmt.Printf("Include patterns: %s\n", viper.GetString("include"))
	}
	if viper.GetString("exclude") != "" {
		fmt.Printf("Exclude patterns: %s\n", viper.GetString("exclude"))
	}
	if viper.GetString("events") != "" {
		fmt.Printf("Include events: %s\n", viper.GetString("events"))
	}
	if viper.GetString("ignore") != "" {
		fmt.Printf("Ignore events: %s\n", viper.GetString("ignore"))
	}
	if viper.GetBool("filter-dev") {
		fmt.Printf("Development filtering: enabled\n")
	}

	// Print webhook information if specified
	if viper.GetString("webhook-url") != "" {
		fmt.Printf("Webhook URL: %s\n", viper.GetString("webhook-url"))
		fmt.Printf("Webhook method: %s\n", viper.GetString("webhook-method"))
		if viper.GetString("webhook-headers") != "" {
			fmt.Printf("Webhook headers: %s\n", viper.GetString("webhook-headers"))
		}
		fmt.Printf("Webhook timeout: %v\n", viper.GetDuration("webhook-timeout"))
		if viper.GetDuration("webhook-debounce-duration") > 0 {
			fmt.Printf("Webhook debounce duration: %v\n", viper.GetDuration("webhook-debounce-duration"))
		}
		fmt.Printf("Webhook max retries: %d\n", viper.GetInt("webhook-max-retries"))
	}

	fmt.Printf("Press Ctrl+C to exit\n\n")

	// Start the event server
	blink.EventServer(
		watchPath,
		viper.GetString("allowed-origin"),
		viper.GetString("event-addr"),
		viper.GetString("event-path"),
		viper.GetDuration("refresh"),
		options...,
	)

	// Block forever (the event server runs in a goroutine)
	select {}
}

// parseHeaders parses a string of headers in the format "key1:value1,key2:value2"
func parseHeaders(headersStr string) map[string]string {
	headers := make(map[string]string)

	// Split by comma
	parts := strings.Split(headersStr, ",")

	for _, part := range parts {
		// Split by colon
		keyValue := strings.SplitN(part, ":", 2)
		if len(keyValue) == 2 {
			key := strings.TrimSpace(keyValue[0])
			value := strings.TrimSpace(keyValue[1])
			headers[key] = value
		}
	}

	return headers
}
