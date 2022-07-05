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
	"errors"
	"os"

	pb "github.com/infinimesh/proto/node"
	accpb "github.com/infinimesh/proto/node/accounts"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
)

func makeConnection(ctx context.Context) (*grpc.ClientConn, error) {
	var opts []grpc.DialOption
	if insec := viper.GetBool("insecure"); insec {
		opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	} else {
		opts = append(opts, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{
			InsecureSkipVerify: true,
		})))
	}

	return grpc.DialContext(ctx, viper.GetString("infinimesh"), opts...)
}

// make AccountsServiceClient
func makeAccountsServiceClient(ctx context.Context) (pb.AccountsServiceClient, error) {
	conn, err := makeConnection(ctx)
	if err != nil {
		return nil, err
	}
	return pb.NewAccountsServiceClient(conn), nil
}

// make context with bearer token metadata
func makeContextWithBearerToken() context.Context {
	token := viper.GetString("token")
	if token == "" {
		return context.Background()
	}
	return metadata.AppendToOutgoingContext(context.Background(), "authorization", "Bearer "+token)
}

// accountsCmd represents the accounts command
var accountsCmd = &cobra.Command{
	Use:     "accounts",
	Short:   "Manage infinimesh Accounts",
	Aliases: []string{"acc", "accs", "account"},
	RunE:    listAccountsCmd.RunE,
}

func PrintAccountsPool(pool []*accpb.Account) {
	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	t.AppendHeader(table.Row{"UUID", "Title", "Enabled", "Default NS"})

	rows := make([]table.Row, len(pool))
	for i, acc := range pool {
		rows[i] = table.Row{acc.Uuid, acc.Title, acc.Enabled, acc.DefaultNamespace}
	}
	t.AppendRows(rows)

	t.SortBy([]table.SortBy{
		{Name: "UUID", Mode: table.Asc},
	})

	t.AppendFooter(table.Row{"Total Found", len(pool)})
	t.Render()
}

var listAccountsCmd = &cobra.Command{
	Use:     "list",
	Short:   "List infinimesh Accounts",
	Aliases: []string{"ls", "l"},
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := makeContextWithBearerToken()
		client, err := makeAccountsServiceClient(ctx)
		if err != nil {
			return err
		}

		r, err := client.List(ctx, &pb.EmptyMessage{})
		if err != nil {
			return err
		}

		if printJson, _ := cmd.Flags().GetBool("json"); printJson {
			return printJsonResponse(r)
		}

		PrintAccountsPool(r.Accounts)
		return nil
	},
}

var getAccountCmd = &cobra.Command{
	Use:   "get",
	Short: "Get infinimesh Account",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := makeContextWithBearerToken()
		client, err := makeAccountsServiceClient(ctx)
		if err != nil {
			return err
		}

		r, err := client.Get(ctx, &accpb.Account{Uuid: args[0]})
		if err != nil {
			return err
		}

		if printJson, _ := cmd.Flags().GetBool("json"); printJson {
			return printJsonResponse(r)
		}

		PrintAccountsPool([]*accpb.Account{r})
		return nil
	},
}

var createAccountCmd = &cobra.Command{
	Use:   "create [namespace] [username] [login]",
	Short: "Create infinimesh Account",
	Long: `Create infinimesh Account
Args:
	[namespace] - infinimesh Namespace UUID
	[username] - new Account Title
	[login] - new Account Login(part of credentials)
`,
	Aliases: []string{"crt"},
	Args:    cobra.ExactArgs(3),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := makeContextWithBearerToken()
		client, err := makeAccountsServiceClient(ctx)
		if err != nil {
			return err
		}

		ns := args[0]
		uname := args[1]
		username := args[2]

		enabled, _ := cmd.Flags().GetBool("enable")

		var cred *accpb.Credentials
		password, err := cmd.Flags().GetString("password")
		if err != nil {
			return err
		}
		if password != "" {
			cred = &accpb.Credentials{
				Type: "standard",
				Data: []string{username, password},
			}
		}

		key, err := cmd.Flags().GetString("ldap-provider-key")
		if err != nil {
			return err
		}
		if key != "" {
			cred = &accpb.Credentials{
				Type: "ldap",
				Data: []string{username, key},
			}
		}

		if cred == nil {
			return errors.New("no Password or LDAP Provider Key given")
		}

		r, err := client.Create(ctx, &accpb.CreateRequest{
			Account: &accpb.Account{
				Title:   uname,
				Enabled: enabled,
			},
			Credentials: cred,
			Namespace:   ns,
		})
		if err != nil {
			return err
		}

		if printJson, _ := cmd.Flags().GetBool("json"); printJson {
			return printJsonResponse(r)
		}

		PrintAccountsPool([]*accpb.Account{r.Account})
		return nil
	},
}

func init() {
	createAccountCmd.Flags().BoolP("enable", "e", false, "Enable Account upon create")
	createAccountCmd.Flags().String("password", "", "Account Password for standard Credentials")
	createAccountCmd.Flags().String("ldap-provider-key", "", "LDAP Provider Key for LDAP Credentials")

	accountsCmd.AddCommand(getAccountCmd)
	accountsCmd.AddCommand(listAccountsCmd)
	accountsCmd.AddCommand(createAccountCmd)
	rootCmd.AddCommand(accountsCmd)
}
