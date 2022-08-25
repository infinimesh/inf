/*
Copyright © 2021-2022 Nikita Ivanovski info@slnt-opp.xyz

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
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"github.com/Pallinder/go-randomdata"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var devCmd = &cobra.Command{
	Use:     "dev",
	Aliases: []string{"developer"},
	Short:   "Kit for infinimesh Developers",
}

var genUIPluginUrlCmd = &cobra.Command{
	Use:   "ui-plug-url [plugin-entrypoint (defaults to http://localhost:3000/e)]",
	Short: "Generates URL as if infinemsh Console would (run help to read more)",
	Long: `infinemesh Console (Web UI) has Plugins support. Which are basically embedded iFrames.
	To Make Plugin work without additional systems and make it actually Embedded into rest of the UX flow, pluging receive a certain set of data.
	
	Which is:
		- User JWT Token (token)
		- User Title (title)
		- Currently selected Namespace (namespace)
		- Currently selected Theme(dark / light) (theme)
		- infinimesh REST API Base URL (api)
	`,
	RunE: func(cmd *cobra.Command, args []string) error {

		params := make(map[string]string)

		entrypoint := "http://localhost:3000/#/e"
		if len(args) > 0 {
			entrypoint = args[0]
		}

		api, _ := cmd.Flags().GetString("api")
		params["api"] = api

		token, _ := cmd.Flags().GetString("token")
		if token == "" {
			token = viper.GetString("token")
			if token == "" {
				fmt.Println("[WARN] No token is given or found in CLI store, using placeholder!")
				token = "aSBhbSBub3QgYQ.SldUIHRva2Vu.anVzdCBob2xkaW5nIHBsYWNlIGhlcmU"
			}
		}
		params["token"] = token

		namespace, _ := cmd.Flags().GetString("namespace")
		params["namespace"] = namespace
		title, _ := cmd.Flags().GetString("title")
		params["title"] = title

		params["theme"] = "dark"
		if l, _ := cmd.Flags().GetBool("light"); l {
			params["theme"] = "light"
		}

		infos := true
		if urlOnly, _ := cmd.Flags().GetBool("url-only"); urlOnly {
			infos = false
		}

		if infos {
			fmt.Println("Resulting parameters:")
			fmt.Printf("\tEntrypoint: %s\n", entrypoint)
			for k, v := range params {
				fmt.Printf("\t%s: %s\n", k, v)
			}
		}

		query_json, err := json.Marshal(params)
		if err != nil {
			return err
		}

		query := base64.StdEncoding.EncodeToString(query_json)
		url := fmt.Sprintf("%s?a=%s", entrypoint, query)

		if _json, _ := cmd.Flags().GetBool("json"); _json {
			r, err := json.MarshalIndent(map[string]string{
				"query": query,
				"url":   url,
			}, "", "  ")
			if err != nil {
				return err
			}
			fmt.Println(r)
			return nil
		}

		fmt.Printf("Raw Query: %s\n", query)
		fmt.Printf("Plugin iFrame URL: %s\n", url)

		return nil
	},
}

func init() {

	initConfig()

	api := &url.URL{
		Scheme: "http",
		Host:   "api.infinimesh.local",
	}

	if g := viper.GetString("infinimesh"); g != "" {
		if !viper.GetBool("insecure") {
			api.Scheme = "https"
		}

		api.Host = strings.Split(g, ":")[0]
	}

	genUIPluginUrlCmd.Flags().String("token", "", "Token to encode into URL (defaults to CLI stored auth)")
	genUIPluginUrlCmd.Flags().String("title", randomdata.SillyName(), "User Title to encode (defaults to random)")
	genUIPluginUrlCmd.Flags().String("namespace", "infinimesh", "Namespace to encode")
	genUIPluginUrlCmd.Flags().Bool("light", false, "Wether to encode Light theme (Dark is used by default)")
	genUIPluginUrlCmd.Flags().String("api", api.String(), "infinimesh REST API Base URL to use")
	genUIPluginUrlCmd.Flags().Bool("url-only", false, "Print resulting URL only")

	devCmd.AddCommand(genUIPluginUrlCmd)
	rootCmd.AddCommand(devCmd)
}
