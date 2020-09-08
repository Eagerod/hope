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

import (
	"github.com/Eagerod/hope/pkg/ssh"
)

var cfgFile string
var configParseError error
var debugLogFlag bool
var verboseLogFlag bool

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
	rootCmd.AddCommand(configCmd)

	initConfigCmdFlags()

	log.Debug("Executing:", os.Args)
	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}

func init() {
	cobra.OnInitialize(initConfig, initLogger)

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.hope.yaml)")
	rootCmd.PersistentFlags().BoolVar(&debugLogFlag, "debug", false, "set the log level to debug; ignoring otherwise configured log levels")
	rootCmd.PersistentFlags().BoolVar(&verboseLogFlag, "verbose", false, "set the log level to verbose; ignoring otherwise configured log levels")
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

	if verboseLogFlag {
		log.SetLevel(log.TraceLevel)
	} else if debugLogFlag {
		log.SetLevel(log.DebugLevel)
	} else {
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
	}
	log.SetOutput(os.Stdout)

	if failed {
		log.Info("Failed to parse loglevel. Defaulting to INFO")
	} else {
		log.Trace("Set log level to ", log.GetLevel())
	}

	if configParseError != nil {
		log.Error(configParseError)
	}

	log.Debug("Using config file:", viper.ConfigFileUsed())

	// Replace some pkg functions with logging enabled versions of themselves.
	originalSSHExec := ssh.ExecSSH
	ssh.ExecSSH = func(args ...string) error {
		log.Debug("ssh", args)
		return originalSSHExec(args...)
	}

	originalSSHGet := ssh.GetSSH
	ssh.GetSSH = func(args ...string) (string, error) {
		log.Debug("ssh", args)
		return originalSSHGet(args...)
	}

	originalSCPExec := ssh.ExecSCP
	ssh.ExecSCP = func(args ...string) error {
		log.Debug("scp", args)
		return originalSCPExec(args...)
	}
}
