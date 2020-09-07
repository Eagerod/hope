package cmd

import (
	"fmt"
	"os"
)

import (
	homedir "github.com/mitchellh/go-homedir"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var cfgFile string
var configParseError error

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "hope",
	Short: "command line tool for managing all the resources I have deployed at home",
	Long: `Hope is a command line tool that has been set up to manage all the manual pieces
of managing my home Kubernetes cluster. It includes mechanisms for setting up 
my router, my switch (maybe, eventually), and controlling the management of the
Kubernetes resources I run.`,
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(addCmd)
	rootCmd.AddCommand(setCmd)
	rootCmd.AddCommand(deployCmd)
	rootCmd.AddCommand(versionCmd)

	log.Debug("Executing:", os.Args)
	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}

func init() {
	cobra.OnInitialize(initConfig, initLogger)

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.hope.yaml)")
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := homedir.Dir()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		// Search config in home directory with name ".hope" (without extension).
		viper.AddConfigPath(home)
		viper.SetConfigName(".hope")
	}

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	configParseError = viper.ReadInConfig()
}

func initLogger() {
	failed := false

	switch logLevel := viper.GetString("loglevel"); logLevel {
	case "trace":
	case "verbose":
		log.SetLevel(log.TraceLevel)
	case "debug":
		log.SetLevel(log.DebugLevel)
	case "info":
		log.SetLevel(log.InfoLevel)
	case "warn":
	case "warning":
		log.SetLevel(log.WarnLevel)
	case "error":
		log.SetLevel(log.ErrorLevel)
	default:
		log.SetLevel(log.InfoLevel)
		failed = true
	}

	log.SetOutput(os.Stdout)

	if failed {
		log.Info("Failed to parse loglevel. Defaulting to INFO")
	}

	if configParseError != nil {
		log.Error(configParseError)
	}

	log.Debug("Using config file:", viper.ConfigFileUsed())
}
