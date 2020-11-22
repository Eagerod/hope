package cmd

import (
	"fmt"
	"os"
	"strings"
)

import (
	homedir "github.com/mitchellh/go-homedir"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

import (
	"github.com/Eagerod/hope/cmd/hope/unifi"

	"github.com/Eagerod/hope/pkg/docker"
	"github.com/Eagerod/hope/pkg/envsubst"
	"github.com/Eagerod/hope/pkg/kubeutil"
	"github.com/Eagerod/hope/pkg/scp"
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
	rootCmd.AddCommand(deployCmd)
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(hostnameCmd)
	rootCmd.AddCommand(kubeconfigCmd)
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(removeCmd)
	rootCmd.AddCommand(resetCmd)
	rootCmd.AddCommand(runCmd)
	rootCmd.AddCommand(shellCmd)
	rootCmd.AddCommand(sshCmd)
	rootCmd.AddCommand(tokenCmd)

	rootCmd.AddCommand(unifi.RootCommand)

	initDeployCmdFlags()
	initHostnameCmdFlags()
	initKubeconfigCmdFlags()
	initListCmdFlags()
	initRemoveCmdFlags()
	initResetCmd()
	initRunCmdFlags()
	initShellCmd()
	initSshCmd()
	initTokenCmd()

	unifi.InitUnifiCommand()

	log.Debug("Executing:", os.Args)
	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}

func init() {
	cobra.OnInitialize(initConfig, initLogger, patchInvocations)

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
		case "trace", "verbose":
			log.SetLevel(log.TraceLevel)
		case "debug":
			log.SetLevel(log.DebugLevel)
		case "info":
			log.SetLevel(log.InfoLevel)
		case "warn", "warning":
			log.SetLevel(log.WarnLevel)
		case "error":
			log.SetLevel(log.ErrorLevel)
		default:
			log.SetLevel(log.InfoLevel)
			failed = true
		}
	}
	log.SetOutput(os.Stderr)

	if failed {
		log.Info("Failed to parse loglevel. Defaulting to INFO")
	} else {
		log.Trace("Set log level to ", log.GetLevel())
	}

	if configParseError != nil {
		log.Error(configParseError)
	}

	log.Debug("Using config file: ", viper.ConfigFileUsed())
}

func patchInvocations() {
	// TODO: Probably create a single os.Exec wrapper of some kind, cause this
	//   is getting ridiculous.
	oldExecDocker := docker.ExecDocker
	docker.ExecDocker = func(args ...string) error {
		if docker.UseSudo {
			log.Debug("sudo docker ", strings.Join(args, " "))
		} else {
			log.Debug("docker ", strings.Join(args, " "))
		}
		return oldExecDocker(args...)
	}

	oldEnvsubstBytes := envsubst.GetEnvsubstBytes
	envsubst.GetEnvsubstBytes = func(args []string, contents []byte) ([]byte, error) {
		argsKeys := []string{}
		for _, key := range args {
			argsKeys = append(argsKeys, fmt.Sprintf("$%s", key))
		}

		log.Debug("echo **(", len(contents), " chars)** | envsubst ", strings.Join(argsKeys, ","))
		return oldEnvsubstBytes(args, contents)
	}

	oldEnvsubstBytesArgs := envsubst.GetEnvsubstBytesArgs
	envsubst.GetEnvsubstBytesArgs = func(args map[string]string, contents []byte) ([]byte, error) {
		argsKeys := []string{}
		for key, _ := range args {
			argsKeys = append(argsKeys, fmt.Sprintf("$%s", key))
		}

		log.Debug("echo **(", len(contents), " chars)** | envsubst ", strings.Join(argsKeys, ","))
		return oldEnvsubstBytesArgs(args, contents)
	}

	oldExecKubectl := kubeutil.ExecKubectl
	kubeutil.ExecKubectl = func(kubectl *kubeutil.Kubectl, args ...string) error {
		log.Debug("kubectl ", strings.Join(args, " "))
		return oldExecKubectl(kubectl, args...)
	}

	oldGetKubectl := kubeutil.GetKubectl
	kubeutil.GetKubectl = func(kubectl *kubeutil.Kubectl, args ...string) (string, error) {
		log.Debug("kubectl ", strings.Join(args, " "))
		return oldGetKubectl(kubectl, args...)
	}

	oldInKubectl := kubeutil.InKubectl
	kubeutil.InKubectl = func(kubectl *kubeutil.Kubectl, stdin string, args ...string) error {
		log.Debug("echo **(", len(stdin), " chars)** | kubectl ", strings.Join(args, " "))
		return oldInKubectl(kubectl, stdin, args...)
	}

	oldExecScp := scp.ExecSCP
	scp.ExecSCP = func(args ...string) error {
		log.Debug("scp ", strings.Join(args, " "))
		return oldExecScp(args...)
	}

	oldExecSsh := ssh.ExecSSH
	ssh.ExecSSH = func(args ...string) error {
		log.Debug("ssh ", strings.Join(args, " "))
		return oldExecSsh(args...)
	}

	oldGetSsh := ssh.GetSSH
	ssh.GetSSH = func(args ...string) (string, error) {
		log.Debug("ssh ", strings.Join(args, " "))
		return oldGetSsh(args...)
	}

	oldGetErrorSsh := ssh.GetErrorSSH
	ssh.GetErrorSSH = func(args ...string) (string, error) {
		log.Debug("ssh ", strings.Join(args, " "))
		return oldGetErrorSsh(args...)
	}
}
