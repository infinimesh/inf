/*
Copyright © 2022 Infinite Devices GmbH, Nikita Ivanovski info@slnt-opp.xyz

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"

	"github.com/spf13/viper"
)

var cfgFile string
var infContext string

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "inf",
	Short: "infinimesh Platform CLI",
	RunE:  contextCmd.RunE,
}

var VERSION string

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if VERSION == "" {
		VERSION = "dev"
	}
	cobra.CheckErr(rootCmd.Execute())
}

func init() {
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.default.infinimesh.yaml)")
	rootCmd.PersistentFlags().StringVar(&infContext, "context", "", "Use a specific config context (default is default)")
	rootCmd.PersistentFlags().Bool("json", false, "Print output as json")
	rootCmd.PersistentFlags().Bool("verbose", false, "Print additional info related to the CLI itself")

	cobra.OnInitialize(initConfig)
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		loadProfile()
	}

	viper.AutomaticEnv() // read in environment variables that match

	verbose, _ := rootCmd.Flags().GetBool("verbose")
	// If a config file is found, read it in.
	err := viper.ReadInConfig()
	if err == nil && verbose {
		fmt.Println("Using context: ", infContext)
		fmt.Println("Using config file:", viper.ConfigFileUsed())
	}
}

func loadProfile() {
	checkProfile()

	home, err := os.UserHomeDir()
	cobra.CheckErr(err)

	// Search config in home directory with name ".inf" (without extension).
	viper.AddConfigPath(home)
	viper.SetConfigType("yaml")

	config_name := fmt.Sprintf(".%s.infinimesh", infContext)

	viper.SetConfigName(config_name)

	cfgFile = fmt.Sprintf("%s/%s.yaml", home, config_name)

	if _, err := os.Stat(cfgFile); os.IsNotExist(err) {
		if _, err := os.Create(cfgFile); err != nil { // perm 0666
			fmt.Fprintln(os.Stderr, "Can't create default config file")
			panic(err)
		}
	}
}

func checkProfile() {
	if infContext != "" {
		return
	}

	home, err := os.UserHomeDir()
	cobra.CheckErr(err)

	faulty_conf := false
	profilesCfg := fmt.Sprintf("%s/.infinimesh.contexts", home)
	if _, err := os.Stat(profilesCfg); os.IsNotExist(err) {
		faulty_conf = true
		if _, err := os.Create(profilesCfg); err != nil { // perm 0666
			fmt.Fprintln(os.Stderr, "Can't create default contexts config file")
			panic(err)
		}
	}

	contexts := struct {
		Selected string `yaml:"selected"`
	}{
		Selected: "default",
	}
	profilesBytes, err := os.ReadFile(profilesCfg)
	if err != nil || len(profilesBytes) == 0 {
		faulty_conf = true
	}
	if err == nil {
		if yaml.Unmarshal(profilesBytes, &contexts) != nil {
			faulty_conf = true
		}
	}

	infContext = contexts.Selected

	if faulty_conf {
		r, _ := yaml.Marshal(contexts)
		os.WriteFile(profilesCfg, r, 0640)
	}
}

func printJsonResponse(data interface{}) error {
	bytes, err := json.Marshal(data)
	if err != nil {
		return err
	}
	fmt.Println(string(bytes))
	return nil
}
