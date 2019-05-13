package cmd

import (
	"os"

	homedir "github.com/mitchellh/go-homedir"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	cfgFile    string
	debug      bool
	maxStreams int
	since      string
	until      string
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:          "loro",
	Short:        "Loro Only Repeats Output",
	SilenceUsage: true,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		if viper.GetBool("debug") {
			log.SetLevel(log.DebugLevel)
		}
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(-1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "Config file (default is $HOME/.loro.yaml)")
	rootCmd.PersistentFlags().BoolVar(&debug, "debug", false, "Enable debug logs")
	listCmd.PersistentFlags().StringVarP(&prefix, "prefix", "p", "", "Stream Name or prefix")
	rootCmd.PersistentFlags().StringVarP(&since, "since", "s", "1h", "Fetch logs since timestamp (e.g. 2013-01-02T13:23:37), relative (e.g. 42m for 42 minutes), or all for all logs")
	rootCmd.PersistentFlags().StringVarP(&until, "until", "u", "now", "Fetch logs until timestamp (e.g. 2013-01-02T13:23:37) or relative (e.g. 42m for 42 minutes)")
	rootCmd.PersistentFlags().IntVarP(&maxStreams, "max-streams", "m", 50, "Maximum number of streams to fetch from (for prefix search)")
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
			log.Fatal(err)
			os.Exit(1)
		}

		// Search config in home directory with name ".loro" (without extension).
		viper.AddConfigPath(home)
		viper.SetConfigName(".loro")
	}

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		log.Info("Using config file:", viper.ConfigFileUsed())
	}
}
