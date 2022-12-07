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
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"

	"github.com/manifoldco/promptui"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"

	pb "github.com/infinimesh/proto/node"
	accpb "github.com/infinimesh/proto/node/accounts"
)

func getVersion() string {
	return VERSION
}

// contextCmd represents the context command
var contextCmd = &cobra.Command{
	Use:     "context",
	Aliases: []string{"ctx"},
	Short:   "Print current infinimesh CLI Context | Aliases: ctx",
	RunE: func(cmd *cobra.Command, args []string) error {
		data := make(map[string]interface{})
		data["version"] = getVersion()

		data["host"] = viper.Get("infinimesh")
		if data["host"] == nil {
			data = map[string]interface{}{
				"error": "No infinimesh context found",
			}
		}

		if insec := viper.GetBool("insecure"); insec {
			data["insecure"] = insec
		}

		if printJson, _ := cmd.Flags().GetBool("json"); printJson {
			return printJsonResponse(data)
		}

		caser := cases.Title(language.English)
		for k, v := range data {
			fmt.Printf("%s: %v\n", caser.String(k), v)
		}

		return nil
	},
}

var loginCmd = &cobra.Command{
	Use:     "login",
	Aliases: []string{"l", "auth", "a"},
	Short:   "Authorize in infinimesh and store credentials",
	RunE: func(cmd *cobra.Command, args []string) error {
		creds := credentials.NewTLS(&tls.Config{InsecureSkipVerify: true})
		insec, _ := cmd.Flags().GetBool("insecure")
		if insec {
			creds = insecure.NewCredentials()
		}
		conn, err := grpc.Dial(args[0], grpc.WithTransportCredentials(creds))
		if err != nil {
			return err
		}

		client := pb.NewAccountsServiceClient(conn)

		t := "standard"
		if ok, err := cmd.Flags().GetBool("ldap"); err != nil {
			return err
		} else if ok {
			t = "ldap"
		}

		var password string
		if p, _ := cmd.Flags().GetString("password"); p != "" {
			password = p
		} else {

			prompt := promptui.Prompt{
				Label: "Password",
				Mask:  '*',
			}

			password, err = prompt.Run()
			if err != nil {
				return err
			}
		}

		req := &pb.TokenRequest{
			Auth: &accpb.Credentials{
				Type: t, Data: []string{args[1], password},
			},
		}

		res, err := client.Token(context.Background(), req)
		if err != nil {
			return err
		}
		token := res.GetToken()
		printToken, _ := cmd.Flags().GetBool("print-token")
		if printToken {
			fmt.Println(token)
		}

		viper.Set("infinimesh", args[0])
		viper.Set("token", token)
		viper.Set("insecure", insec)

		err = viper.WriteConfig()
		return err
	},
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print infinimesh CLI version",
	RunE: func(cmd *cobra.Command, args []string) error {
		if printJson, _ := cmd.Flags().GetBool("json"); printJson {
			data, err := json.Marshal(map[string]string{
				"version": getVersion(),
			})
			if err != nil {
				return err
			}
			fmt.Println(string(data))
			return nil
		}

		fmt.Println("CLI Version:", getVersion())
		return nil
	},
}

func init() {
	loginCmd.Flags().StringP("password", "p", "", "Password for Standard Credentials")
	loginCmd.Flags().Bool("print-token", false, "")
	loginCmd.Flags().Bool("insecure", false, "Use WithInsecure instead of TLS")
	loginCmd.Flags().Bool("ldap", false, "Use Credentials Type LDAP")

	rootCmd.AddCommand(contextCmd)
	rootCmd.AddCommand(loginCmd)
	rootCmd.AddCommand(versionCmd)
}
